package oidcavatar

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxAvatarSize = 5 << 20

type Profile struct {
	Name             string
	PreferredUsername string
	GivenName        string
	FamilyName       string
	Picture          string
}

func (p Profile) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	if p.PreferredUsername != "" {
		return p.PreferredUsername
	}
	full := strings.TrimSpace(p.GivenName + " " + p.FamilyName)
	if full != "" {
		return full
	}
	return ""
}

func FetchAvatar(pictureURL, storageDir string, userID int64, logger *slog.Logger) (string, error) {
	parsed, err := url.Parse(pictureURL)
	if err != nil {
		return "", fmt.Errorf("invalid picture URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return "", fmt.Errorf("only HTTPS picture URLs are allowed")
	}

	host := parsed.Hostname()
	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		return "", fmt.Errorf("DNS lookup failed for %s: %w", host, err)
	}
	for _, ip := range ips {
		if isPrivateIP(ip.IP) {
			return "", fmt.Errorf("picture URL resolves to private IP")
		}
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			if req.URL.Scheme != "https" {
				return fmt.Errorf("redirect to non-HTTPS URL")
			}
			return nil
		},
	}

	resp, err := client.Get(pictureURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch avatar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("avatar fetch returned status %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	ext, ok := avatarExtension(ct)
	if !ok {
		return "", fmt.Errorf("unsupported content-type: %s", ct)
	}

	filename := fmt.Sprintf("oidc-%d-%d%s", userID, time.Now().UnixNano(), ext)
	relativePath := filepath.Join("avatars", filename)
	absolutePath := filepath.Join(storageDir, relativePath)

	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return "", fmt.Errorf("failed to prepare avatar directory: %w", err)
	}

	file, err := os.Create(absolutePath)
	if err != nil {
		return "", fmt.Errorf("failed to create avatar file: %w", err)
	}

	limited := io.LimitReader(resp.Body, maxAvatarSize+1)
	n, err := io.Copy(file, limited)
	if closeErr := file.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(absolutePath)
		return "", fmt.Errorf("failed to write avatar file: %w", err)
	}
	if n > maxAvatarSize {
		_ = os.Remove(absolutePath)
		return "", fmt.Errorf("avatar exceeds %d byte limit", maxAvatarSize)
	}

	return relativePath, nil
}

func RemoveFile(storageDir, relativePath string) {
	if relativePath == "" {
		return
	}
	abs := filepath.Join(storageDir, filepath.Clean(relativePath))
	safe := filepath.Clean(filepath.Join(storageDir, "avatars"))
	if !strings.HasPrefix(abs, safe) {
		return
	}
	_ = os.Remove(abs)
}

func avatarExtension(contentType string) (string, bool) {
	ct := strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])
	switch ct {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		{mustParseCIDR("fd00::/8")},
	}
	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}
	return false
}

func mustParseCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}
