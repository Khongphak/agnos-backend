package handler

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"github.com/agnos-assessment/agnos-backend/internal/service"
	internalws "github.com/agnos-assessment/agnos-backend/internal/ws"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WSHandler handles websocket upgrade requests for patient and staff connections.
type WSHandler struct {
	hub       *internalws.Hub
	svc       service.PatientService
	jwtSecret []byte
}

func NewWSHandler(hub *internalws.Hub, svc service.PatientService, jwtSecret string) *WSHandler {
	return &WSHandler{hub: hub, svc: svc, jwtSecret: []byte(jwtSecret)}
}

// HandlePatient upgrades to WS, validates hospital_code, then reads patient form updates.
// Close code 1008 (policy violation) is sent when hospital_code is not found.
func (h *WSHandler) HandlePatient(c *gin.Context) {
	code := c.Query("hospital_code")
	if code == "" {
		c.JSON(http.StatusBadRequest, response.NewError("INVALID_INPUT", "hospital_code is required"))
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	hospital, err := h.svc.FindHospitalByCode(c.Request.Context(), code)
	if err != nil {
		conn.WriteMessage( //nolint:errcheck
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "hospital not found"),
		)
		conn.Close() //nolint:errcheck
		return
	}

	sessionID := newSessionUUID()
	ack, _ := json.Marshal(map[string]string{"type": "connected", "session_id": sessionID})
	if err := conn.WriteMessage(websocket.TextMessage, ack); err != nil {
		conn.Close() //nolint:errcheck
		return
	}

	patient := &internalws.PatientConn{
		Conn:       conn,
		SessionID:  sessionID,
		HospitalID: hospital.ID,
		Hub:        h.hub,
	}
	h.hub.RegisterPatient <- patient
	patient.ReadPump() // blocks until conn closes
}

// HandleStaff validates JWT from ?token=, upgrades to WS, then listens (receive-only).
func (h *WSHandler) HandleStaff(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, response.NewError("UNAUTHORIZED", "token is required"))
		return
	}

	token, err := jwt.ParseWithClaims(tokenStr, &service.StaffClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return h.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, response.NewError("UNAUTHORIZED", "invalid or expired token"))
		return
	}

	claims, ok := token.Claims.(*service.StaffClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.NewError("UNAUTHORIZED", "invalid token claims"))
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	staff := &internalws.StaffConn{
		Conn:       conn,
		HospitalID: claims.HospitalID,
		Hub:        h.hub,
		Send:       make(chan []byte, 64),
	}
	h.hub.RegisterStaff <- staff
	go staff.WritePump()
	staff.ReadPump() // blocks until conn closes
}

// newSessionUUID generates a random UUID v4.
func newSessionUUID() string {
	b := make([]byte, 16)
	rand.Read(b) //nolint:errcheck
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
