package server

import (
    "encoding/json"
    "errors"
    "fmt"
    "log"
    "math"
    "net/http"
    "os"
    "strconv"
    "strings"
    "time"

    "github.com/example/cs2trader/internal/auth"
    "github.com/gorilla/handlers"
    "github.com/gorilla/mux"
    "github.com/gorilla/sessions"
    nats "github.com/nats-io/nats.go"
)

type Server struct {
    router              *mux.Router
    port                int
    corsAllowedOrigins  []string
    sessionStore        *sessions.CookieStore
    steamOpenID         *auth.SteamOpenID
    clientRedirectURL   string
    natsURL             string
    natsConn            *nats.Conn
}

func NewServerFromEnv() (*Server, error) {
    port := 8080
    if v := os.Getenv("API_PORT"); v != "" {
        p, err := strconv.Atoi(v)
        if err != nil {
            return nil, fmt.Errorf("invalid API_PORT: %w", err)
        }
        port = p
    }

    corsOrigins := []string{"*"}
    if v := os.Getenv("CORS_ORIGINS"); v != "" {
        corsOrigins = strings.Split(v, ",")
    }

    sessionSecret := os.Getenv("SESSION_SECRET")
    if sessionSecret == "" {
        return nil, errors.New("SESSION_SECRET is required")
    }
    store := sessions.NewCookieStore([]byte(sessionSecret))
    store.Options = &sessions.Options{Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode}

    steamRealm := os.Getenv("STEAM_OPENID_REALM")
    steamReturnTo := os.Getenv("STEAM_OPENID_RETURN_TO")
    if steamRealm == "" || steamReturnTo == "" {
        return nil, errors.New("STEAM_OPENID_REALM and STEAM_OPENID_RETURN_TO are required")
    }
    steam := auth.NewSteamOpenID(steamRealm, steamReturnTo)

    clientRedirectURL := os.Getenv("CLIENT_REDIRECT_URL")
    if clientRedirectURL == "" {
        clientRedirectURL = "/"
    }

    s := &Server{
        router:             mux.NewRouter(),
        port:               port,
        corsAllowedOrigins: corsOrigins,
        sessionStore:       store,
        steamOpenID:        steam,
        clientRedirectURL:  clientRedirectURL,
        natsURL:            getenvDefault("NATS_URL", "nats://nats:4222"),
    }
    s.registerRoutes()
    return s, nil
}

func (s *Server) registerRoutes() {
    r := s.router
    r.HandleFunc("/healthz", s.handleHealth).Methods(http.MethodGet)
    r.HandleFunc("/auth/steam/login", s.handleSteamLogin).Methods(http.MethodGet)
    r.HandleFunc("/auth/steam/callback", s.handleSteamCallback).Methods(http.MethodGet)
    r.HandleFunc("/auth/me", s.handleAuthMe).Methods(http.MethodGet)
    r.HandleFunc("/market/prices", s.handleMarketPrices).Methods(http.MethodGet)
}

func (s *Server) Start() error {
    // Connect to NATS (best-effort)
    var err error
    s.natsConn, err = nats.Connect(s.natsURL, nats.Timeout(3*time.Second))
    if err != nil {
        log.Printf("warn: failed to connect to NATS at %s: %v", s.natsURL, err)
    } else {
        if _, subErr := s.natsConn.Subscribe("orders", func(msg *nats.Msg) {
            log.Printf("[NATS] orders: %s", string(msg.Data))
        }); subErr != nil {
            log.Printf("warn: failed to subscribe to 'orders': %v", subErr)
        } else {
            log.Printf("connected to NATS at %s and subscribed to 'orders'", s.natsURL)
        }
    }

    cors := handlers.CORS(
        handlers.AllowedOrigins(s.corsAllowedOrigins),
        handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
        handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
        handlers.AllowCredentials(),
    )
    addr := fmt.Sprintf(":%d", s.port)
    log.Printf("API listening on %s", addr)
    return http.ListenAndServe(addr, cors(s.router))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
}

func (s *Server) handleSteamLogin(w http.ResponseWriter, r *http.Request) {
    url := s.steamOpenID.LoginURL()
    http.Redirect(w, r, url, http.StatusFound)
}

func (s *Server) handleSteamCallback(w http.ResponseWriter, r *http.Request) {
    steamID, err := s.steamOpenID.VerifyCallback(r.Context(), r.URL.Query())
    if err != nil {
        http.Error(w, "failed to verify OpenID", http.StatusUnauthorized)
        return
    }

    session, _ := s.sessionStore.Get(r, "session")
    session.Values["steam_id"] = steamID
    _ = session.Save(r, w)

    // Redirect to frontend
    redirect := s.clientRedirectURL
    if strings.Contains(redirect, "?") {
        redirect += "&steam_id=" + steamID
    } else if strings.HasSuffix(redirect, "/") || !strings.Contains(redirect, "?") {
        redirect += "?steam_id=" + steamID
    }
    http.Redirect(w, r, redirect, http.StatusFound)
}

func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
    session, _ := s.sessionStore.Get(r, "session")
    steamID, _ := session.Values["steam_id"].(string)
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]any{
        "authenticated": steamID != "",
        "steam_id":      steamID,
    })
}

func (s *Server) handleMarketPrices(w http.ResponseWriter, r *http.Request) {
    // Return simple synthetic time series
    type point struct {
        Timestamp int64   `json:"timestamp"`
        Price     float64 `json:"price"`
    }
    now := time.Now()
    points := make([]point, 0, 60)
    base := 100.0
    for i := 59; i >= 0; i-- {
        t := now.Add(-time.Duration(i) * time.Minute)
        price := base + 10.0*0.5*(1+math.Sin(float64(i)/6.0))
        points = append(points, point{Timestamp: t.Unix(), Price: price})
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(points)
}

func getenvDefault(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}

// mathSin is a tiny indirection so we do not pull full math import in multiple places
// removed custom sin indirection; using math.Sin

