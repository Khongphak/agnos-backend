package ws

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// ── mockConn ──────────────────────────────────────────────────────────────────

type mockConn struct {
	written  [][]byte
	readCh   chan []byte
	closedCh chan struct{}
	closed   bool
}

func newMockConn() *mockConn {
	return &mockConn{
		readCh:   make(chan []byte, 8),
		closedCh: make(chan struct{}),
	}
}

func (m *mockConn) ReadMessage() (int, []byte, error) {
	select {
	case data := <-m.readCh:
		return 1, data, nil
	case <-m.closedCh:
		return 0, nil, fmt.Errorf("connection closed")
	}
}

func (m *mockConn) WriteMessage(_ int, data []byte) error {
	if m.closed {
		return fmt.Errorf("write to closed conn")
	}
	m.written = append(m.written, data)
	return nil
}

func (m *mockConn) Close() error {
	if !m.closed {
		m.closed = true
		close(m.closedCh)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newRunningHub() *Hub {
	h := NewHub()
	go h.Run()
	return h
}

func tick() { time.Sleep(20 * time.Millisecond) }

// ── tests ─────────────────────────────────────────────────────────────────────

func TestHub_RegisterAndUnregisterPatient(t *testing.T) {
	h := newRunningHub()
	conn := newMockConn()

	patient := &PatientConn{
		Conn:       conn,
		SessionID:  "sess-abc",
		HospitalID: 1,
		Hub:        h,
	}
	h.RegisterPatient <- patient
	tick()

	// unregister; hub should close the conn and broadcast inactive
	h.UnregisterPatient <- patient
	tick()

	if !conn.closed {
		t.Error("expected patient conn to be closed after unregister")
	}
}

func TestHub_BroadcastFormUpdateToStaff(t *testing.T) {
	h := newRunningHub()

	staffSend := make(chan []byte, 4)
	staff := &StaffConn{
		Conn:       newMockConn(),
		HospitalID: 10,
		Hub:        h,
		Send:       staffSend,
	}
	h.RegisterStaff <- staff
	tick() // wait for hub to process registration

	patient := &PatientConn{
		Conn:       newMockConn(),
		SessionID:  "sess-1",
		HospitalID: 10,
		Hub:        h,
	}
	h.RegisterPatient <- patient
	tick() // wait for hub to process registration

	h.InboundPatient <- patientMsg{
		conn: patient,
		payload: IncomingPatientMessage{
			Type:   "form_update",
			Status: "filling",
			Data:   json.RawMessage(`{"field":"name"}`),
		},
	}

	select {
	case raw := <-staffSend:
		var msg OutgoingMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("invalid JSON broadcast: %v", err)
		}
		if msg.Type != "form_update" {
			t.Errorf("expected type 'form_update', got %q", msg.Type)
		}
		if msg.SessionID != "sess-1" {
			t.Errorf("expected session_id 'sess-1', got %q", msg.SessionID)
		}
		if msg.Status != "filling" {
			t.Errorf("expected status 'filling', got %q", msg.Status)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout: staff did not receive broadcast")
	}
}

func TestHub_BroadcastInactiveOnUnregister(t *testing.T) {
	h := newRunningHub()

	staffSend := make(chan []byte, 4)
	staff := &StaffConn{
		Conn:       newMockConn(),
		HospitalID: 5,
		Hub:        h,
		Send:       staffSend,
	}
	h.RegisterStaff <- staff

	patient := &PatientConn{
		Conn:       newMockConn(),
		SessionID:  "sess-disconnect",
		HospitalID: 5,
		Hub:        h,
	}
	h.RegisterPatient <- patient
	tick()

	h.UnregisterPatient <- patient

	select {
	case raw := <-staffSend:
		var msg OutgoingMessage
		json.Unmarshal(raw, &msg) //nolint:errcheck
		if msg.Status != "inactive" {
			t.Errorf("expected status 'inactive', got %q", msg.Status)
		}
		if string(msg.Data) != "null" {
			t.Errorf("expected data null, got %s", msg.Data)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout: staff did not receive inactive broadcast on disconnect")
	}
}

func TestHub_InactivityTimeout(t *testing.T) {
	// Override timeout to something short for testing.
	// We do this by directly firing the inactivityFired channel instead of
	// waiting 30 s; the timer callback just sends to that channel.
	h := newRunningHub()

	staffSend := make(chan []byte, 4)
	staff := &StaffConn{
		Conn:       newMockConn(),
		HospitalID: 3,
		Hub:        h,
		Send:       staffSend,
	}
	h.RegisterStaff <- staff

	patConn := newMockConn()
	patient := &PatientConn{
		Conn:       patConn,
		SessionID:  "sess-timeout",
		HospitalID: 3,
		Hub:        h,
	}
	h.RegisterPatient <- patient
	tick()

	// Simulate timer firing by sending directly to inactivityFired
	h.inactivityFired <- patient

	select {
	case raw := <-staffSend:
		var msg OutgoingMessage
		json.Unmarshal(raw, &msg) //nolint:errcheck
		if msg.Status != "inactive" {
			t.Errorf("expected 'inactive' from timeout, got %q", msg.Status)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout: no inactive broadcast after inactivity timer fired")
	}

	tick()
	if !patConn.closed {
		t.Error("expected patient conn to be closed after inactivity timeout")
	}
}

func TestHub_InactivityTimeout_AlreadyUnregistered(t *testing.T) {
	// Timer fires after a disconnect — hub should silently no-op.
	h := newRunningHub()

	patient := &PatientConn{
		Conn:       newMockConn(),
		SessionID:  "sess-already-gone",
		HospitalID: 1,
		Hub:        h,
	}
	h.RegisterPatient <- patient
	tick()
	h.UnregisterPatient <- patient
	tick()

	// Should not panic or double-broadcast
	h.inactivityFired <- patient
	tick()
}

func TestHub_StaffOnlyReceivesBroadcastForOwnHospital(t *testing.T) {
	h := newRunningHub()

	sendH1 := make(chan []byte, 4)
	sendH2 := make(chan []byte, 4)

	h.RegisterStaff <- &StaffConn{Conn: newMockConn(), HospitalID: 1, Hub: h, Send: sendH1}
	tick()
	h.RegisterStaff <- &StaffConn{Conn: newMockConn(), HospitalID: 2, Hub: h, Send: sendH2}
	tick()

	patient := &PatientConn{
		Conn:       newMockConn(),
		SessionID:  "sess-h1",
		HospitalID: 1,
		Hub:        h,
	}
	h.RegisterPatient <- patient
	tick()

	h.InboundPatient <- patientMsg{
		conn:    patient,
		payload: IncomingPatientMessage{Type: "form_update", Status: "filling", Data: json.RawMessage("null")},
	}

	select {
	case <-sendH1:
		// correct: hospital 1 received
	case <-time.After(200 * time.Millisecond):
		t.Fatal("hospital-1 staff did not receive broadcast")
	}

	select {
	case <-sendH2:
		t.Error("hospital-2 staff should NOT receive broadcast from hospital-1 patient")
	case <-time.After(50 * time.Millisecond):
		// correct: no message for hospital 2
	}
}
