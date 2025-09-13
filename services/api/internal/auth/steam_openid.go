package auth

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "regexp"
)

type SteamOpenID struct {
    Realm   string
    ReturnTo string
}

func NewSteamOpenID(realm, returnTo string) *SteamOpenID {
    return &SteamOpenID{Realm: realm, ReturnTo: returnTo}
}

func (s *SteamOpenID) LoginURL() string {
    params := url.Values{}
    params.Set("openid.ns", "http://specs.openid.net/auth/2.0")
    params.Set("openid.mode", "checkid_setup")
    params.Set("openid.return_to", s.ReturnTo)
    params.Set("openid.realm", s.Realm)
    params.Set("openid.identity", "http://specs.openid.net/auth/2.0/identifier_select")
    params.Set("openid.claimed_id", "http://specs.openid.net/auth/2.0/identifier_select")
    return "https://steamcommunity.com/openid/login?" + params.Encode()
}

var steamIDRegex = regexp.MustCompile(`https://steamcommunity.com/openid/id/(\d+)`)

func (s *SteamOpenID) VerifyCallback(ctx context.Context, q url.Values) (string, error) {
    // Build verification post
    verify := url.Values{}
    for key := range q {
        verify.Set(key, q.Get(key))
    }
    verify.Set("openid.mode", "check_authentication")

    resp, err := http.PostForm("https://steamcommunity.com/openid/login", verify)
    if err != nil {
        return "", fmt.Errorf("verify post failed: %w", err)
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    // Steam returns key:value\n lines with is_valid:true
    if !containsIsValidTrue(string(body)) {
        return "", fmt.Errorf("invalid openid assertion")
    }

    claimed := q.Get("openid.claimed_id")
    matches := steamIDRegex.FindStringSubmatch(claimed)
    if len(matches) != 2 {
        return "", fmt.Errorf("could not parse steam id")
    }
    return matches[1], nil
}

func containsIsValidTrue(s string) bool {
    // naive check
    return regexp.MustCompile(`(?m)^is_valid:true$`).FindStringIndex(s) != nil
}

