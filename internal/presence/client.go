package presence

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Client state values reported to the UI.
const (
	StatusIdle         = "idle"
	StatusConnected    = "connected"
	StatusDisabled     = "disabled"
	StatusUnavailable  = "unavailable"
	StatusReconnecting = "reconnecting"
	StatusError        = "error"
)

const (
	// SetActivityTimeout bounds one SET_ACTIVITY round trip; presence is
	// best-effort and must never block the launch monitor.
	SetActivityTimeout = 2 * time.Second
)

// Retry defaults. The values live on Client so tests can tune one client
// without racing a background retry loop.
const (
	// ReconnectBackoff delays the first reconnection attempt after a drop.
	ReconnectBackoff = 5 * time.Second
	// MaxReconnectAttempts caps background reconnection work.
	MaxReconnectAttempts = 6
)

// ErrDisabled is returned by mutating calls while the client is disabled.
var ErrDisabled = errors.New("discord presence is disabled")

// Client is a lazy, fail-independent Discord IPC connection. Every call is
// safe to make when Discord is absent: it degrades to no-op with a status.
type Client struct {
	ClientID string

	mu           sync.Mutex
	conn         netConn
	status       string
	enabled      bool
	nonce        uint64
	closed       bool
	backoff      time.Duration
	lastActivity *Activity
	retrying     bool
	// retryBackoff/retryAttempts are per-client so tests can tune them.
	retryBackoff  time.Duration
	retryAttempts int
}

// netConn is the subset of net.Conn the client needs; narrowed for tests.
type netConn interface {
	Write(b []byte) (int, error)
	Read(b []byte) (int, error)
	Close() error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// NewClient returns a disabled client for the given application ID.
func NewClient(clientID string) *Client {
	return &Client{
		ClientID:      clientID,
		status:        StatusDisabled,
		retryBackoff:  ReconnectBackoff,
		retryAttempts: MaxReconnectAttempts,
	}
}

// Status reports the current connection state.
func (c *Client) Status() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// Enable marks the client allowed to connect. It does not dial immediately;
// the first SetActivity performs the connection.
func (c *Client) Enable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.enabled = true
	c.status = StatusIdle
	c.backoff = 0
	if c.lastActivity != nil {
		c.scheduleRetryLocked()
	}
}

// Enabled reports whether publishing is allowed.
func (c *Client) Enabled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enabled
}

// Disable closes any live connection and stops publishing.
func (c *Client) Disable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeLocked()
	c.enabled = false
	c.status = StatusDisabled
}

// Close releases the connection for shutdown.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	c.closeLocked()
}

func (c *Client) closeLocked() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

// SetActivity publishes one activity update. When disabled it is a no-op
// returning ErrDisabled; when Discord is unreachable it records a status
// and returns the dial error.
func (c *Client) SetActivity(activity *Activity) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.enabled || c.closed {
		return ErrDisabled
	}
	if activity == nil {
		return c.clearLocked()
	}
	c.lastActivity = activity
	if c.conn == nil {
		if err := c.connectLocked(); err != nil {
			c.scheduleRetryLocked()
			return err
		}
	}
	if err := c.setActivityLocked(activity); err != nil {
		c.scheduleRetryLocked()
		return err
	}
	return nil
}

// connectLocked dials, performs the handshake, and updates status.
func (c *Client) connectLocked() error {
	conn, err := dial()
	if err != nil {
		c.status = StatusUnavailable
		return fmt.Errorf("discord not reachable: %w", err)
	}
	payload, err := json.Marshal(handshakePayload{V: "1", ClientID: c.ClientID})
	if err != nil {
		_ = conn.Close()
		c.status = StatusError
		return err
	}
	if err := c.roundTripLocked(conn, opHandshake, payload); err != nil {
		_ = conn.Close()
		c.status = StatusError
		return err
	}
	c.conn = conn
	c.status = StatusConnected
	c.backoff = 0
	return nil
}

// roundTripLocked writes one frame and reads the matching reply.
func (c *Client) roundTripLocked(conn netConn, opcode int, payload []byte) error {
	_ = conn.SetWriteDeadline(time.Now().Add(SetActivityTimeout))
	_ = conn.SetReadDeadline(time.Now().Add(SetActivityTimeout))
	if _, err := conn.Write(encodeFrame(opcode, payload)); err != nil {
		return err
	}
	reply, err := readFrame(conn)
	if err != nil {
		return err
	}
	if reply.Opcode == opClose {
		return errors.New("discord closed the connection")
	}
	var resp responseData
	if err := json.Unmarshal(reply.Payload, &resp); err != nil {
		return fmt.Errorf("invalid discord reply: %w", err)
	}
	if resp.Evt == "error" {
		return fmt.Errorf("discord rejected the request: %s", resp.Message)
	}
	return nil
}

// setActivityLocked sends one SET_ACTIVITY envelope.
func (c *Client) setActivityLocked(activity *Activity) error {
	c.nonce++
	nonce := fmt.Sprintf("plume-%d", c.nonce)
	args := struct {
		PID      int       `json:"pid"`
		Activity *Activity `json:"activity"`
	}{PID: activity.PID, Activity: activity}
	payload, err := json.Marshal(envelope{Cmd: "SET_ACTIVITY", Args: args, Nonce: nonce})
	if err != nil {
		return err
	}
	if err := c.roundTripLocked(c.conn, opFrame, payload); err != nil {
		c.closeLocked()
		c.status = StatusReconnecting
		return err
	}
	c.status = StatusConnected
	return nil
}

// clearLocked removes the current presence. It only dials when a connection
// already exists, so clearing after a failed publish never reconnects.
func (c *Client) clearLocked() error {
	c.lastActivity = nil
	if c.conn == nil {
		return nil
	}
	empty := &Activity{PID: 0}
	if err := c.setActivityLocked(empty); err != nil {
		c.closeLocked()
		c.status = StatusIdle
		return err
	}
	// After clearing, drop the connection so Discord shows no stale state.
	c.closeLocked()
	c.status = StatusIdle
	return nil
}

// scheduleRetryLocked starts one background retry loop, if none is running.
// Callers must hold c.mu.
func (c *Client) scheduleRetryLocked() {
	if c.retrying || c.closed || !c.enabled || c.lastActivity == nil {
		return
	}
	c.retrying = true
	activity := c.lastActivity
	go c.retryLoop(activity)
}

// retryLoop re-publishes the last activity with bounded backoff. It stops on
// success, on Disable/Close, or when the attempt budget is exhausted.
func (c *Client) retryLoop(activity *Activity) {
	defer func() {
		c.mu.Lock()
		c.retrying = false
		c.mu.Unlock()
	}()
	c.mu.Lock()
	delay := c.retryBackoff
	attempts := c.retryAttempts
	c.mu.Unlock()
	for attempt := 0; attempt < attempts; attempt++ {
		time.Sleep(delay)
		c.mu.Lock()
		if c.closed || !c.enabled {
			c.mu.Unlock()
			return
		}
		if c.conn == nil {
			if err := c.connectLocked(); err != nil {
				c.mu.Unlock()
				if delay < 30*time.Second {
					delay *= 2
				}
				continue
			}
		}
		if err := c.setActivityLocked(activity); err != nil {
			c.mu.Unlock()
			if delay < 30*time.Second {
				delay *= 2
			}
			continue
		}
		c.mu.Unlock()
		return
	}
	c.mu.Lock()
	if c.conn == nil && c.enabled && !c.closed {
		c.status = StatusUnavailable
	}
	c.mu.Unlock()
}
