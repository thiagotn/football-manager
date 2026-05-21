package services

import (
	"context"
	"fmt"

	twilio "github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/verify/v2"
)

// TwilioService wraps Twilio Verify for OTP send/check.
type TwilioService struct {
	client     *twilio.RestClient
	verifySID  string
	bypassCode string
	isProd     bool
}

func NewTwilioService(accountSID, authToken, verifySID, bypassCode string, isProd bool) *TwilioService {
	var client *twilio.RestClient
	if accountSID != "" && authToken != "" {
		client = twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: accountSID,
			Password: authToken,
		})
	}
	return &TwilioService{
		client:     client,
		verifySID:  verifySID,
		bypassCode: bypassCode,
		isProd:     isProd,
	}
}

func (s *TwilioService) isBypassActive() bool {
	return s.bypassCode != "" && !s.isProd
}

func (s *TwilioService) isConfigured() bool {
	return s.client != nil && s.verifySID != ""
}

func (s *TwilioService) SendOTP(_ context.Context, whatsapp string) error {
	if s.isBypassActive() {
		return nil
	}
	if !s.isConfigured() {
		return fmt.Errorf("twilio not configured")
	}
	params := &twilioApi.CreateVerificationParams{}
	params.SetTo(whatsapp)
	params.SetChannel("sms")
	_, err := s.client.VerifyV2.CreateVerification(s.verifySID, params)
	return err
}

func (s *TwilioService) CheckOTP(_ context.Context, whatsapp, code string) (bool, error) {
	if s.isBypassActive() {
		return code == s.bypassCode, nil
	}
	if !s.isConfigured() {
		return false, nil
	}
	params := &twilioApi.CreateVerificationCheckParams{}
	params.SetTo(whatsapp)
	params.SetCode(code)
	check, err := s.client.VerifyV2.CreateVerificationCheck(s.verifySID, params)
	if err != nil {
		return false, nil
	}
	return check.Status != nil && *check.Status == "approved", nil
}
