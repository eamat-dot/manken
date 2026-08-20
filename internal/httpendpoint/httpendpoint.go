// Package httpendpoint は、取得元に依存しないHTTP endpoint検証を提供する
package httpendpoint

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Validate は、HTTPまたはHTTPSの絶対endpoint URLを検証する
func Validate(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("endpoint must be an absolute HTTP URL: %w", err)
	}
	return validateStructure(parsed)
}

// ValidateHTTPSOrLoopback は、HTTPSまたはloopback HTTPのendpoint URLを検証する
func ValidateHTTPSOrLoopback(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("endpoint must be an absolute HTTP URL: %w", err)
	}
	if err := validateSchemeAndHost(parsed); err != nil {
		return err
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) {
		return errors.New("endpoint must use https unless its host is loopback")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return errors.New("endpoint must not include user information, query, or fragment")
	}
	return nil
}

// validateStructure は、endpoint URLのscheme、host、禁止する構成要素を検証する
func validateStructure(parsed *url.URL) error {
	if err := validateSchemeAndHost(parsed); err != nil {
		return err
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return errors.New("endpoint must not include user information, query, or fragment")
	}
	return nil
}

// validateSchemeAndHost は、endpoint URLのschemeとhostを検証する
func validateSchemeAndHost(parsed *url.URL) error {
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("endpoint must use http or https and include a host")
	}
	return nil
}

// isLoopbackHost は、名前解決をせずにhostがlocalhostまたはloopback IPか判定する
func isLoopbackHost(host string) bool {
	return strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback()
}
