package ws

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

const inactivityTimeout = 30 * time.Second

// Conn abstracts a websocket connection, allowing mock implementations in tests.
type Conn interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	Close() error
}

// IncomingPatientMessage is what a patient sends over the websocket.
type IncomingPatientMessage struct {
	Type   string          `json:"type"`
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// OutgoingMessage is what staff receives; matches the cross-project contract.
type OutgoingMessage struct {
	Type      string          `json:"type"`       // always "form_update"
	SessionID string          `json:"session_id"` // patient session UUID
	Status    string          `json:"status"`     // filling | submitted | inactive
	Data      json.RawMessage `json:"data"`       // null when inactive
	Timestamp time.Time       `json:"timestamp"`
}

// PatientConn holds a patient's websocket connection state.
type PatientConn struct {
	Conn       Conn
	SessionID  string
	HospitalID int64
	Hub        *Hub
}

// ReadPump reads patient messages and forwards them to the hub. Blocking.
// The caller must register the conn with hub before calling ReadPump.
func (c *PatientConn) ReadPump() {
	defer func() {
		c.Hub.UnregisterPatient <- c
	}()
	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		var msg IncomingPatientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type != "form_update" {
			continue
		}
		c.Hub.InboundPatient <- patientMsg{conn: c, payload: msg}
	}
}

// StaffConn holds a staff member's websocket connection state.
type StaffConn struct {
	Conn       Conn
	HospitalID int64
	Hub        *Hub
	Send       chan []byte
}

// ReadPump reads from the websocket (disconnect detection only). Blocking.
// It closes Send when it exits, which causes WritePump to drain and exit.
func (c *StaffConn) ReadPump() {
	defer func() {
		c.Hub.UnregisterStaff <- c
		close(c.Send)
	}()
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

// WritePump writes messages from the Send channel to the websocket.
func (c *StaffConn) WritePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}
