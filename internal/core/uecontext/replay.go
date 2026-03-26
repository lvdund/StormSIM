package uecontext

import (
	"encoding/base64"
	"fmt"
	"log"
	"stormsim/pkg/model"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

var Capture *MessageCapture
var Replay *ReplayEngine

func (ue *UeContext) triggerReplay() {
	defer ue.wg.Done()

	for _, entry := range Replay.session.Messages {
		i := 0
		for entry.StateID != ue.state_mm.CurrentState() {
			if i >= 10 {
				ue.Warn("[Fuzz] Force state %d\n", entry.StateID)
				ue.state_mm.ForceSetState(entry.StateID)
				break
			}
			i++
			time.Sleep(100 * time.Millisecond)
		}

		ue.sendNas(entry.DataByte)
	}
}

type MessageType string

const (
	MessageFromUe MessageType = "from-ue"
	MessageToUe   MessageType = "to-ue"
)

// CapturedMessage represents a single captured message
type CapturedMessage struct {
	Timestamp   time.Time       `yaml:"timestamp"`
	Type        MessageType     `yaml:"type"`
	StateID     model.StateType `yaml:"state_id,omitempty"` // Only for messages from ue
	MsgID       int             `yaml:"msg_id,omitempty"`   // Only for messages to ue
	DataString  string          `yaml:"data,omitempty"`     // Base64 encoded []byte for ue send
	DataByte    []byte          `yaml:"-"`
	SequenceNum int             `yaml:"sequence_num"`
}

type CaptureSession struct {
	SessionID string            `yaml:"session_id"`
	StartTime time.Time         `yaml:"start_time"`
	EndTime   time.Time         `yaml:"end_time"`
	Messages  []CapturedMessage `yaml:"messages"`
	Metadata  map[string]string `yaml:"metadata,omitempty"`
}

type MessageCapture struct {
	session     *CaptureSession
	sequenceNum int
}

func NewMessageCapture(sessionID string) *MessageCapture {
	return &MessageCapture{
		session: &CaptureSession{
			SessionID: sessionID,
			StartTime: time.Now(),
			Messages:  make([]CapturedMessage, 0),
			Metadata:  make(map[string]string),
		},
		sequenceNum: 0,
	}
}

func (mc *MessageCapture) CaptureMsgFromUe(stateID model.StateType, data []byte) {
	mc.sequenceNum++

	encodedData := base64.StdEncoding.EncodeToString(data)

	msg := CapturedMessage{
		Timestamp:   time.Now(),
		Type:        MessageFromUe,
		StateID:     stateID,
		DataString:  encodedData,
		SequenceNum: mc.sequenceNum,
	}

	mc.session.Messages = append(mc.session.Messages, msg)
}

func (mc *MessageCapture) CaptureMsgToUe(msgID int) {
	mc.sequenceNum++

	msg := CapturedMessage{
		Timestamp:   time.Now(),
		Type:        MessageToUe,
		MsgID:       msgID,
		SequenceNum: mc.sequenceNum,
	}

	mc.session.Messages = append(mc.session.Messages, msg)
}

func (mc *MessageCapture) AddMetadata(key, value string) {
	mc.session.Metadata[key] = value
}

func (mc *MessageCapture) SaveToFile(filename string) error {
	mc.session.EndTime = time.Now()

	data, err := yaml.Marshal(mc.session)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Capture session saved to %s", filename)
	return nil
}

// ReplayEngine handles replaying captured sessions
type ReplayEngine struct {
	session *CaptureSession
}

func LoadFromFile(filename string) (*ReplayEngine, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var session CaptureSession
	err = yaml.Unmarshal(data, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &ReplayEngine{session: &session}, nil
}

func (re *ReplayEngine) getMessages() []CapturedMessage {
	messages := make([]CapturedMessage, len(re.session.Messages))

	for i, msg := range re.session.Messages {
		replayMsg := CapturedMessage{
			Type:        msg.Type,
			StateID:     msg.StateID,
			MsgID:       msg.MsgID,
			SequenceNum: msg.SequenceNum,
			Timestamp:   msg.Timestamp,
		}

		// Decode base64 data back to []byte for A->B messages
		if msg.Type == MessageFromUe && msg.DataString != "" {
			data, err := base64.StdEncoding.DecodeString(msg.DataString)
			if err != nil {
				log.Printf("Warning: failed to decode message data: %v", err)
			} else {
				replayMsg.DataByte = data
			}
		}

		messages[i] = replayMsg
	}

	return messages
}

func (re *ReplayEngine) ReplayWithCallback(callback func(msg CapturedMessage)) {
	messages := re.getMessages()

	log.Printf("Starting replay of %d messages from session %s",
		len(messages), re.session.SessionID)

	for _, msg := range messages {
		callback(msg)
	}

	log.Printf("Replay completed")
}

func (re *ReplayEngine) ReplayWithTiming(callback func(msg CapturedMessage)) {
	messages := re.getMessages()
	if len(messages) == 0 {
		return
	}

	log.Printf("Starting timed replay of %d messages", len(messages))

	startTime := messages[0].Timestamp

	for _, msg := range messages {
		// Calculate delay from start
		delay := msg.Timestamp.Sub(startTime)
		time.Sleep(delay)

		callback(msg)
	}

	log.Printf("Timed replay completed")
}
