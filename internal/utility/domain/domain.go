package domain

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/pkg/output"
)

var httpClient = &http.Client{}

// DNSRecord is an LLM-friendly DNS record
type DNSRecord struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

// DNSResult is the DNS lookup result
type DNSResult struct {
	Domain  string      `json:"domain"`
	Records []DNSRecord `json:"records"`
}

// WhoisInfo is LLM-friendly WHOIS information
type WhoisInfo struct {
	Domain      string `json:"domain"`
	Registrar   string `json:"registrar,omitempty"`
	Created     string `json:"created,omitempty"`
	Updated     string `json:"updated,omitempty"`
	Expires     string `json:"expires,omitempty"`
	Status      string `json:"status,omitempty"`
	NameServers string `json:"nameservers,omitempty"`
	Registrant  string `json:"registrant,omitempty"`
	Country     string `json:"country,omitempty"`
}

// SSLInfo is LLM-friendly SSL certificate information
type SSLInfo struct {
	Domain    string   `json:"domain"`
	Issuer    string   `json:"issuer"`
	Subject   string   `json:"subject"`
	NotBefore string   `json:"valid_from"`
	NotAfter  string   `json:"valid_until"`
	DaysLeft  int      `json:"days_left"`
	DNSNames  []string `json:"dns_names,omitempty"`
	IsValid   bool     `json:"is_valid"`
	Version   int      `json:"version"`
	Serial    string   `json:"serial"`
	SigAlgo   string   `json:"sig_algo"`
	IsExpired bool     `json:"is_expired"`
	IsNotYet  bool     `json:"is_not_yet_valid"`
}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "domain",
		Aliases: []string{"dns", "whois", "ssl"},
		Short:   "Domain utilities (DNS, WHOIS, SSL)",
	}

	cmd.AddCommand(newDNSCmd())
	cmd.AddCommand(newWhoisCmd())
	cmd.AddCommand(newSSLCmd())

	return cmd
}

func newDNSCmd() *cobra.Command {
	var recordType string

	cmd := &cobra.Command{
		Use:   "dns [domain]",
		Short: "DNS lookup (A, AAAA, MX, TXT, NS records)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := cleanDomain(args[0])

			types := []string{"A", "AAAA", "MX", "TXT", "NS"}
			if recordType != "" {
				types = []string{strings.ToUpper(recordType)}
			}

			var allRecords []DNSRecord
			for _, t := range types {
				records, err := dnsLookup(domain, t)
				if err != nil {
					continue
				}
				allRecords = append(allRecords, records...)
			}

			if len(allRecords) == 0 {
				return output.PrintError("not_found", "No DNS records found for: "+domain, nil)
			}

			result := DNSResult{
				Domain:  domain,
				Records: allRecords,
			}

			return output.Print(result)
		},
	}

	cmd.Flags().StringVarP(&recordType, "type", "t", "", "Record type (A, AAAA, MX, TXT, NS)")

	return cmd
}

func newWhoisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whois [domain]",
		Short: "WHOIS lookup for domain registration info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := cleanDomain(args[0])

			// API requires name and suffix as separate parameters
			parts := strings.SplitN(domain, ".", 2)
			var reqURL string
			if len(parts) == 2 {
				reqURL = fmt.Sprintf("https://whois.freeaiapi.xyz/?name=%s&suffix=%s",
					url.QueryEscape(parts[0]), url.QueryEscape(parts[1]))
			} else {
				reqURL = fmt.Sprintf("https://whois.freeaiapi.xyz/?name=%s", url.QueryEscape(domain))
			}

			resp, err := doRequest(reqURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			var data map[string]any

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				return output.PrintError("parse_failed", err.Error(), nil)
			}

			// Check if the API returned an error
			if status := getString(data, "status"); status == "error" {
				errMsg := getString(data, "message")
				if errMsg == "" {
					errMsg = "WHOIS lookup failed"
				}
				return output.PrintError("whois_error", errMsg,
					map[string]string{"domain": domain})
			}

			info := WhoisInfo{
				Domain:      domain,
				Registrar:   getString(data, "registrar"),
				Created:     getString(data, "creation_date"),
				Updated:     getString(data, "updated_date"),
				Expires:     getString(data, "expiration_date"),
				Status:      getString(data, "status"),
				NameServers: getString(data, "name_servers"),
				Registrant:  getString(data, "registrant"),
				Country:     getString(data, "country"),
			}

			// Try alternate field names used by different WHOIS APIs
			if info.Created == "" {
				info.Created = getString(data, "creation_datetime")
			}
			if info.Expires == "" {
				info.Expires = getString(data, "expiry_datetime")
			}
			if info.Updated == "" {
				info.Updated = getString(data, "updatedDate")
			}
			if info.NameServers == "" {
				info.NameServers = getString(data, "nameServers")
			}

			return output.Print(info)
		},
	}

	return cmd
}

func newSSLCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "ssl [domain]",
		Short: "SSL certificate information for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := cleanDomain(args[0])

			addr := fmt.Sprintf("%s:%d", domain, port)

			conn, err := tls.Dial("tcp", addr, &tls.Config{
				InsecureSkipVerify: true, // We want to inspect even invalid certs
			})
			if err != nil {
				return output.PrintError("connection_failed", "Could not connect to "+addr+": "+err.Error(), nil)
			}
			defer conn.Close()

			certs := conn.ConnectionState().PeerCertificates
			if len(certs) == 0 {
				return output.PrintError("no_cert", "No certificate found for: "+domain, nil)
			}

			cert := certs[0]
			now := time.Now()

			daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
			isExpired := now.After(cert.NotAfter)
			isNotYet := now.Before(cert.NotBefore)

			info := SSLInfo{
				Domain:    domain,
				Issuer:    cert.Issuer.CommonName,
				Subject:   cert.Subject.CommonName,
				NotBefore: cert.NotBefore.Format(time.RFC3339),
				NotAfter:  cert.NotAfter.Format(time.RFC3339),
				DaysLeft:  daysLeft,
				DNSNames:  cert.DNSNames,
				IsValid:   !isExpired && !isNotYet,
				Version:   cert.Version,
				Serial:    cert.SerialNumber.String(),
				SigAlgo:   cert.SignatureAlgorithm.String(),
				IsExpired: isExpired,
				IsNotYet:  isNotYet,
			}

			return output.Print(info)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 443, "Port to connect to")

	return cmd
}

func dnsLookup(domain, recordType string) ([]DNSRecord, error) {
	reqURL := fmt.Sprintf("https://dns.google/resolve?name=%s&type=%s",
		url.QueryEscape(domain), url.QueryEscape(recordType))

	resp, err := doRequest(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Status   int `json:"Status"`
		Question []struct {
			Name string `json:"name"`
			Type int    `json:"type"`
		} `json:"Question"`
		Answer []struct {
			Name string `json:"name"`
			Type int    `json:"type"`
			TTL  int    `json:"TTL"`
			Data string `json:"data"`
		} `json:"Answer"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Status != 0 || len(data.Answer) == 0 {
		return nil, fmt.Errorf("no records found")
	}

	var records []DNSRecord
	for _, ans := range data.Answer {
		records = append(records, DNSRecord{
			Type:  dnsTypeToString(ans.Type),
			Name:  strings.TrimSuffix(ans.Name, "."),
			Value: ans.Data,
			TTL:   ans.TTL,
		})
	}

	return records, nil
}

func dnsTypeToString(t int) string {
	types := map[int]string{
		1:  "A",
		2:  "NS",
		5:  "CNAME",
		6:  "SOA",
		15: "MX",
		16: "TXT",
		28: "AAAA",
	}
	if s, ok := types[t]; ok {
		return s
	}
	return fmt.Sprintf("TYPE%d", t)
}

func cleanDomain(domain string) string {
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimSuffix(domain, "/")
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	return domain
}

func getString(data map[string]any, key string) string {
	if v, ok := data[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		case []any:
			if len(val) > 0 {
				var parts []string
				for _, item := range val {
					if s, ok := item.(string); ok {
						parts = append(parts, s)
					}
				}
				return strings.Join(parts, ", ")
			}
		}
	}
	return ""
}

func doRequest(reqURL string) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, output.PrintError("fetch_failed", err.Error(), nil)
	}

	req.Header.Set("User-Agent", "Pocket-CLI/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, output.PrintError("fetch_failed", err.Error(), nil)
	}

	if resp.StatusCode == 429 {
		resp.Body.Close()
		return nil, output.PrintError("rate_limited", "Rate limit exceeded, try again later", nil)
	}

	if resp.StatusCode >= 400 {
		resp.Body.Close()
		return nil, output.PrintError("fetch_failed", fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	return resp, nil
}
