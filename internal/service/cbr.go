package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
)

type CBRClient struct {
	httpClient *http.Client
	margin     float64
}

func NewCBRClient(margin float64) *CBRClient {
	return &CBRClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		margin: margin,
	}
}

func (c *CBRClient) KeyRate(ctx context.Context) (float64, error) {
	reqBody := buildSOAPRequest()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx",
		bytes.NewBufferString(reqBody),
	)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	req.Header.Set("SOAPAction", "http://web.cbr.ru/KeyRate")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("cbr request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("cbr bad status: %s", resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	rate, err := parseKeyRate(raw)
	if err != nil {
		return 0, err
	}

	return rate + c.margin, nil
}

func buildSOAPRequest() string {
	fromDate := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	toDate := time.Now().Format("2006-01-02")

	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<soap12:Envelope xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">
  <soap12:Body>
    <KeyRate xmlns="http://web.cbr.ru/">
      <fromDate>%s</fromDate>
      <ToDate>%s</ToDate>
    </KeyRate>
  </soap12:Body>
</soap12:Envelope>`, fromDate, toDate)
}

func parseKeyRate(raw []byte) (float64, error) {
	doc := etree.NewDocument()

	if err := doc.ReadFromBytes(raw); err != nil {
		return 0, err
	}

	rateText, ok := findTagText(doc.Root(), "Rate")
	if !ok {
		return 0, errors.New("key rate not found")
	}

	rateText = strings.ReplaceAll(strings.TrimSpace(rateText), ",", ".")

	rate, err := strconv.ParseFloat(rateText, 64)
	if err != nil {
		return 0, fmt.Errorf("parse key rate: %w", err)
	}

	return rate, nil
}

func findTagText(el *etree.Element, tag string) (string, bool) {
	if el == nil {
		return "", false
	}

	if strings.EqualFold(el.Tag, tag) {
		return el.Text(), true
	}

	for _, child := range el.ChildElements() {
		if text, ok := findTagText(child, tag); ok {
			return text, true
		}
	}

	return "", false
}
