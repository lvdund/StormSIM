package logger

import (
	"testing"
	"time"
)

func TestRingBuffer_PushAndUpdate(t *testing.T) {
	rb := NewRingBuffer[string](5)

	idx1, gen1 := rb.Push("entry1")
	idx2, gen2 := rb.Push("entry2")

	if idx1 != 0 || idx2 != 1 {
		t.Errorf("expected indices 0, 1, got %d, %d", idx1, idx2)
	}
	if gen1 == gen2 {
		t.Error("generations should be different")
	}

	if !rb.Update(idx1, gen1, "updated1") {
		t.Error("update should succeed with correct generation")
	}

	entries := rb.GetAll()
	if entries[0] != "updated1" {
		t.Errorf("expected updated1, got %s", entries[0])
	}
}

func TestRingBuffer_UpdateFailsAfterOverwrite(t *testing.T) {
	rb := NewRingBuffer[string](2)

	idx1, gen1 := rb.Push("entry1")
	rb.Push("entry2")
	rb.Push("entry3")

	if rb.Update(idx1, gen1, "should_fail") {
		t.Error("update should fail after slot is overwritten")
	}
}

func TestRingBuffer_UpdateFailsWithWrongGeneration(t *testing.T) {
	rb := NewRingBuffer[string](5)

	idx, gen := rb.Push("entry1")

	if rb.Update(idx, gen+1, "should_fail") {
		t.Error("update should fail with wrong generation")
	}
}

func TestDelayTracker_SendLogsImmediatelyWithUnknown(t *testing.T) {
	dt := NewDelayTracker(10)

	dt.RecordSend("nas", "RegistrationRequest")

	logs := dt.GetLogs(0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]
	if entry.RequestType != "RegistrationRequest" {
		t.Errorf("expected RegistrationRequest, got %s", entry.RequestType)
	}
	if entry.ResponseType != "Unknown" {
		t.Errorf("expected Unknown response, got %s", entry.ResponseType)
	}
	if entry.DelayMs != 0 {
		t.Errorf("expected 0 delay, got %f", entry.DelayMs)
	}
}

func TestDelayTracker_ReceiveUpdatesEntry(t *testing.T) {
	dt := NewDelayTracker(10)

	dt.RecordSend("nas", "RegistrationRequest")
	time.Sleep(10 * time.Millisecond)
	dt.RecordReceive("nas", "AuthenticationRequest")

	logs := dt.GetLogs(0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]
	if entry.RequestType != "RegistrationRequest" {
		t.Errorf("expected RegistrationRequest, got %s", entry.RequestType)
	}
	if entry.ResponseType != "AuthenticationRequest" {
		t.Errorf("expected AuthenticationRequest, got %s", entry.ResponseType)
	}
	if entry.DelayMs < 10 {
		t.Errorf("expected delay >= 10ms, got %f", entry.DelayMs)
	}
}

func TestDelayTracker_AllNasPairs(t *testing.T) {
	testCases := []struct {
		name     string
		request  string
		response string
	}{
		{"Registration->Auth", "RegistrationRequest", "AuthenticationRequest"},
		{"Registration->Identity", "RegistrationRequest", "IdentityRequest"},
		{"Registration->Reject", "RegistrationRequest", "RegistrationReject"},
		{"Identity->Auth", "IdentityResponse", "AuthenticationRequest"},
		{"AuthResponse->Security", "AuthenticationResponse", "SecurityModeCommand"},
		{"SecurityComplete->Accept", "SecurityModeComplete", "RegistrationAccept"},
		{"RegComplete->ConfigUpdate", "RegistrationComplete", "ConfigurationUpdateCommand"},
		{"PDU->Accept", "PduSessionEstablishmentRequest", "PduSessionEstablishmentAccept"},
		{"PDU->Reject", "PduSessionEstablishmentRequest", "PduSessionEstablishmentReject"},
		{"Dereg->Accept", "DeregistrationRequestFromUE", "DeregistrationAcceptFromUE"},
		{"Service->Accept", "ServiceRequest", "ServiceAccept"},
		{"Service->Reject", "ServiceRequest", "ServiceReject"},
		{"AuthFail->AuthReq", "AuthenticationFailure", "AuthenticationRequest"},
		{"AuthFail->AuthReject", "AuthenticationFailure", "AuthenticationReject"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dt := NewDelayTracker(10)

			dt.RecordSend("nas", tc.request)
			dt.RecordReceive("nas", tc.response)

			logs := dt.GetLogs(0)
			if len(logs) != 1 {
				t.Fatalf("expected 1 log entry, got %d", len(logs))
			}

			entry := logs[0]
			if entry.RequestType != tc.request {
				t.Errorf("expected request %s, got %s", tc.request, entry.RequestType)
			}
			if entry.ResponseType != tc.response {
				t.Errorf("expected response %s, got %s", tc.response, entry.ResponseType)
			}
		})
	}
}

func TestDelayTracker_MismatchedResponseSetsUnknownRequest(t *testing.T) {
	dt := NewDelayTracker(10)

	dt.RecordSend("nas", "RegistrationRequest")
	dt.RecordReceive("nas", "ServiceAccept")

	logs := dt.GetLogs(0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]
	if entry.RequestType != "Unknown" {
		t.Errorf("expected Unknown request for mismatch, got %s", entry.RequestType)
	}
	if entry.ResponseType != "ServiceAccept" {
		t.Errorf("expected ServiceAccept, got %s", entry.ResponseType)
	}
}

func TestDelayTracker_ReceiveWithNoPendingPushesUnknown(t *testing.T) {
	dt := NewDelayTracker(10)

	dt.RecordReceive("nas", "AuthenticationRequest")

	logs := dt.GetLogs(0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]
	if entry.RequestType != "Unknown" {
		t.Errorf("expected Unknown request, got %s", entry.RequestType)
	}
	if entry.ResponseType != "AuthenticationRequest" {
		t.Errorf("expected AuthenticationRequest, got %s", entry.ResponseType)
	}
}

func TestDelayTracker_SecurityModeRejectHasNoResponse(t *testing.T) {
	dt := NewDelayTracker(10)

	dt.RecordSend("nas", "SecurityModeReject")

	logs := dt.GetLogs(0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]
	if entry.RequestType != "SecurityModeReject" {
		t.Errorf("expected SecurityModeReject, got %s", entry.RequestType)
	}
	if entry.ResponseType != "Unknown" {
		t.Errorf("expected Unknown response (no expected response), got %s", entry.ResponseType)
	}
}

func TestDelayTracker_RegistrationCompleteHasNoResponse(t *testing.T) {
	dt := NewDelayTracker(10)

	dt.RecordSend("nas", "RegistrationComplete")

	logs := dt.GetLogs(0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]
	if entry.RequestType != "RegistrationComplete" {
		t.Errorf("expected RegistrationComplete, got %s", entry.RequestType)
	}
	if entry.ResponseType != "Unknown" {
		t.Errorf("expected Unknown response, got %s", entry.ResponseType)
	}
}

func TestDelayTracker_MultipleSendsBeforeReceive(t *testing.T) {
	dt := NewDelayTracker(10)

	dt.RecordSend("nas", "RegistrationRequest")
	time.Sleep(5 * time.Millisecond)
	dt.RecordSend("nas", "IdentityResponse")

	logs := dt.GetLogs(0)
	if len(logs) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(logs))
	}

	dt.RecordReceive("nas", "AuthenticationRequest")

	logs = dt.GetLogs(0)
	if len(logs) != 2 {
		t.Fatalf("expected still 2 log entries after receive, got %d", len(logs))
	}

	var foundMatch bool
	for _, entry := range logs {
		if entry.ResponseType == "AuthenticationRequest" && entry.RequestType != "Unknown" {
			foundMatch = true
			break
		}
	}
	if !foundMatch {
		t.Error("expected one entry to be updated with AuthenticationRequest response")
	}
}

func TestDelayTracker_NonNasProtocolStillWorks(t *testing.T) {
	dt := NewDelayTracker(10)

	dt.RecordSend("ngap", "InitialUEMessage")
	time.Sleep(5 * time.Millisecond)
	dt.RecordReceive("ngap", "DownlinkNASTransport")

	logs := dt.GetLogsByProtocol("ngap", 0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 ngap log entry, got %d", len(logs))
	}

	entry := logs[0]
	if entry.RequestType != "InitialUEMessage" {
		t.Errorf("expected InitialUEMessage, got %s", entry.RequestType)
	}
	if entry.ResponseType != "DownlinkNASTransport" {
		t.Errorf("expected DownlinkNASTransport, got %s", entry.ResponseType)
	}
}

func TestDelayTracker_DlNasTransportIsIgnored(t *testing.T) {
	dt := NewDelayTracker(10)

	dt.RecordSend("nas", "RegistrationRequest")
	dt.RecordReceive("nas", "DlNasTransport")

	logs := dt.GetLogs(0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(logs))
	}

	entry := logs[0]
	if entry.ResponseType != "Unknown" {
		t.Errorf("expected Unknown (DlNasTransport ignored), got %s", entry.ResponseType)
	}
}

func TestDelayTracker_SequentialPairing(t *testing.T) {
	dt := NewDelayTracker(10)

	dt.RecordSend("nas", "RegistrationRequest")
	dt.RecordReceive("nas", "AuthenticationRequest")
	dt.RecordSend("nas", "AuthenticationResponse")
	dt.RecordReceive("nas", "SecurityModeCommand")
	dt.RecordSend("nas", "SecurityModeComplete")
	dt.RecordReceive("nas", "RegistrationAccept")
	dt.RecordSend("nas", "RegistrationComplete")
	dt.RecordReceive("nas", "ConfigurationUpdateCommand")

	logs := dt.GetLogs(0)
	if len(logs) != 4 {
		t.Fatalf("expected 4 log entries (including final send), got %d", len(logs))
	}

	expected := []struct {
		req  string
		resp string
	}{
		{"RegistrationRequest", "AuthenticationRequest"},
		{"AuthenticationResponse", "SecurityModeCommand"},
		{"SecurityModeComplete", "RegistrationAccept"},
		{"RegistrationComplete", "ConfigurationUpdateCommand"},
	}

	for i, exp := range expected {
		if logs[i].RequestType != exp.req {
			t.Errorf("entry %d: expected request %s, got %s", i, exp.req, logs[i].RequestType)
		}
		if logs[i].ResponseType != exp.resp {
			t.Errorf("entry %d: expected response %s, got %s", i, exp.resp, logs[i].ResponseType)
		}
	}
}
