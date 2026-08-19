package ws

import (
	"encoding/json"
	"time"
)

type patientMsg struct {
	conn    *PatientConn
	payload IncomingPatientMessage
}

// Hub manages all active patient and staff websocket connections.
// All state mutations happen in the Run() goroutine — no mutex needed.
type Hub struct {
	patients        map[string]*PatientConn
	staffByHospital map[int64]map[*StaffConn]struct{}
	timers          map[string]*time.Timer // session_id -> inactivity timer

	RegisterPatient   chan *PatientConn
	UnregisterPatient chan *PatientConn
	RegisterStaff     chan *StaffConn
	UnregisterStaff   chan *StaffConn
	InboundPatient    chan patientMsg
	inactivityFired   chan *PatientConn
}

func NewHub() *Hub {
	return &Hub{
		patients:          make(map[string]*PatientConn),
		staffByHospital:   make(map[int64]map[*StaffConn]struct{}),
		timers:            make(map[string]*time.Timer),
		RegisterPatient:   make(chan *PatientConn, 8),
		UnregisterPatient: make(chan *PatientConn, 8),
		RegisterStaff:     make(chan *StaffConn, 8),
		UnregisterStaff:   make(chan *StaffConn, 8),
		InboundPatient:    make(chan patientMsg, 64),
		inactivityFired:   make(chan *PatientConn, 8),
	}
}

// Run processes all hub events. Must be called in exactly one goroutine.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.RegisterPatient:
			h.patients[conn.SessionID] = conn
			h.timers[conn.SessionID] = time.AfterFunc(inactivityTimeout, func() {
				h.inactivityFired <- conn
			})

		case conn := <-h.UnregisterPatient:
			if _, ok := h.patients[conn.SessionID]; !ok {
				break
			}
			if t := h.timers[conn.SessionID]; t != nil {
				t.Stop()
			}
			delete(h.timers, conn.SessionID)
			delete(h.patients, conn.SessionID)
			h.broadcastInactive(conn.SessionID, conn.HospitalID)
			conn.Conn.Close() //nolint:errcheck

		case conn := <-h.RegisterStaff:
			if h.staffByHospital[conn.HospitalID] == nil {
				h.staffByHospital[conn.HospitalID] = make(map[*StaffConn]struct{})
			}
			h.staffByHospital[conn.HospitalID][conn] = struct{}{}

		case conn := <-h.UnregisterStaff:
			delete(h.staffByHospital[conn.HospitalID], conn)

		case msg := <-h.InboundPatient:
			if _, ok := h.patients[msg.conn.SessionID]; !ok {
				break
			}
			if t := h.timers[msg.conn.SessionID]; t != nil {
				t.Reset(inactivityTimeout)
			}
			h.broadcastFormUpdate(msg.conn.SessionID, msg.conn.HospitalID, msg.payload)

		case conn := <-h.inactivityFired:
			if _, ok := h.patients[conn.SessionID]; !ok {
				break // already unregistered (e.g., disconnect beat the timer)
			}
			delete(h.timers, conn.SessionID)
			delete(h.patients, conn.SessionID)
			h.broadcastInactive(conn.SessionID, conn.HospitalID)
			conn.Conn.Close() //nolint:errcheck
		}
	}
}

func (h *Hub) broadcastFormUpdate(sessionID string, hospitalID int64, msg IncomingPatientMessage) {
	out := OutgoingMessage{
		Type:      "form_update",
		SessionID: sessionID,
		Status:    msg.Status,
		Data:      msg.Data,
		Timestamp: time.Now().UTC(),
	}
	data, _ := json.Marshal(out)
	h.sendToStaff(hospitalID, data)
}

func (h *Hub) broadcastInactive(sessionID string, hospitalID int64) {
	out := OutgoingMessage{
		Type:      "form_update",
		SessionID: sessionID,
		Status:    "inactive",
		Data:      json.RawMessage("null"),
		Timestamp: time.Now().UTC(),
	}
	data, _ := json.Marshal(out)
	h.sendToStaff(hospitalID, data)
}

func (h *Hub) sendToStaff(hospitalID int64, data []byte) {
	for conn := range h.staffByHospital[hospitalID] {
		select {
		case conn.Send <- data:
		default:
			// slow consumer: skip this message rather than blocking the hub
		}
	}
}
