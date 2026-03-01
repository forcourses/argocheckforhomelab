package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ============================================================
//  Portfolio Tracker — Investment tracking REST API
//
//  Features:
//   - CRUD for portfolios & holdings
//   - Automatic stock price fetching (Alpha Vantage)
//   - P&L calculation per holding and portfolio
//   - Allocation breakdown (% per sector/asset)
//   - Health & readiness endpoints
// ============================================================

type Config struct {
	Port             string
	DatabaseURL      string
	AlphaVantageKey  string
	LogLevel         string
}

type App struct {
	db     *pgxpool.Pool
	config Config
}

// --- Models ---

type Portfolio struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Holding struct {
	ID            int       `json:"id"`
	PortfolioID   int       `json:"portfolio_id"`
	Symbol        string    `json:"symbol"`
	Shares        float64   `json:"shares"`
	AvgCostBasis  float64   `json:"avg_cost_basis"`
	CurrentPrice  float64   `json:"current_price,omitempty"`
	LastPriceAt   time.Time `json:"last_price_at,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type HoldingWithPnL struct {
	Holding
	MarketValue   float64 `json:"market_value"`
	TotalCost     float64 `json:"total_cost"`
	PnL           float64 `json:"pnl"`
	PnLPercent    float64 `json:"pnl_percent"`
	Allocation    float64 `json:"allocation_percent"`
}

type PortfolioSummary struct {
	Portfolio
	TotalValue    float64          `json:"total_value"`
	TotalCost     float64          `json:"total_cost"`
	TotalPnL      float64          `json:"total_pnl"`
	TotalPnLPct   float64          `json:"total_pnl_percent"`
	Holdings      []HoldingWithPnL `json:"holdings"`
}

type StockQuote struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price"`
}

// --- Main ---

func main() {
	cfg := Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://portfolio:portfolio-secret@localhost:5432/portfolio?sslmode=disable"),
		AlphaVantageKey: getEnv("ALPHA_VANTAGE_API_KEY", "demo"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
	}

	// Setup logging
	level, _ := zerolog.ParseLevel(cfg.LogLevel)
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	// Handle subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			runMigrations(cfg)
			return
		case "refresh-prices":
			runPriceRefresh(cfg)
			return
		}
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to ping database")
	}
	log.Info().Msg("Connected to PostgreSQL")

	app := &App{db: pool, config: cfg}

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Health endpoints
	r.Get("/healthz", app.healthz)
	r.Get("/readyz", app.readyz)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(jsonContentType)

		// Portfolios
		r.Route("/portfolios", func(r chi.Router) {
			r.Get("/", app.listPortfolios)
			r.Post("/", app.createPortfolio)
			r.Get("/{id}", app.getPortfolio)
			r.Delete("/{id}", app.deletePortfolio)

			// Holdings within a portfolio
			r.Post("/{id}/holdings", app.addHolding)
			r.Delete("/{id}/holdings/{holdingId}", app.removeHolding)
		})

		// Quotes
		r.Get("/quote/{symbol}", app.getQuote)
	})

	// Swagger-ish docs
	r.Get("/swagger", app.swaggerDocs)
	r.Get("/", app.swaggerDocs)

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Info().Str("port", cfg.Port).Msg("Portfolio Tracker starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

// --- Handlers ---

func (a *App) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (a *App) readyz(w http.ResponseWriter, r *http.Request) {
	if err := a.db.Ping(r.Context()); err != nil {
		http.Error(w, "database not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

func (a *App) listPortfolios(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(),
		`SELECT id, name, description, created_at FROM portfolios ORDER BY created_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch portfolios")
		return
	}
	defer rows.Close()

	var portfolios []Portfolio
	for rows.Next() {
		var p Portfolio
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt); err != nil {
			continue
		}
		portfolios = append(portfolios, p)
	}
	if portfolios == nil {
		portfolios = []Portfolio{}
	}
	writeJSON(w, http.StatusOK, portfolios)
}

func (a *App) createPortfolio(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	var p Portfolio
	err := a.db.QueryRow(r.Context(),
		`INSERT INTO portfolios (name, description) VALUES ($1, $2)
		 RETURNING id, name, description, created_at`,
		req.Name, req.Description).Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create portfolio")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (a *App) getPortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var p Portfolio
	err := a.db.QueryRow(r.Context(),
		`SELECT id, name, description, created_at FROM portfolios WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "Portfolio not found")
		return
	}

	// Fetch holdings with P&L
	rows, err := a.db.Query(r.Context(),
		`SELECT id, portfolio_id, symbol, shares, avg_cost_basis, 
		        COALESCE(current_price, 0), COALESCE(last_price_at, '1970-01-01'), created_at
		 FROM holdings WHERE portfolio_id = $1 ORDER BY symbol`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to fetch holdings")
		return
	}
	defer rows.Close()

	summary := PortfolioSummary{Portfolio: p}
	var totalValue, totalCost float64

	for rows.Next() {
		var h Holding
		if err := rows.Scan(&h.ID, &h.PortfolioID, &h.Symbol, &h.Shares,
			&h.AvgCostBasis, &h.CurrentPrice, &h.LastPriceAt, &h.CreatedAt); err != nil {
			continue
		}

		hp := HoldingWithPnL{Holding: h}
		hp.MarketValue = h.Shares * h.CurrentPrice
		hp.TotalCost = h.Shares * h.AvgCostBasis
		hp.PnL = hp.MarketValue - hp.TotalCost
		if hp.TotalCost > 0 {
			hp.PnLPercent = (hp.PnL / hp.TotalCost) * 100
		}

		totalValue += hp.MarketValue
		totalCost += hp.TotalCost
		summary.Holdings = append(summary.Holdings, hp)
	}

	// Calculate allocations
	for i := range summary.Holdings {
		if totalValue > 0 {
			summary.Holdings[i].Allocation = (summary.Holdings[i].MarketValue / totalValue) * 100
		}
	}

	summary.TotalValue = totalValue
	summary.TotalCost = totalCost
	summary.TotalPnL = totalValue - totalCost
	if totalCost > 0 {
		summary.TotalPnLPct = (summary.TotalPnL / totalCost) * 100
	}

	if summary.Holdings == nil {
		summary.Holdings = []HoldingWithPnL{}
	}
	writeJSON(w, http.StatusOK, summary)
}

func (a *App) deletePortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, err := a.db.Exec(r.Context(), `DELETE FROM portfolios WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) addHolding(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "id")
	var req struct {
		Symbol       string  `json:"symbol"`
		Shares       float64 `json:"shares"`
		AvgCostBasis float64 `json:"avg_cost_basis"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Symbol == "" || req.Shares <= 0 {
		writeError(w, http.StatusBadRequest, "symbol, shares (>0), and avg_cost_basis are required")
		return
	}

	var h Holding
	err := a.db.QueryRow(r.Context(),
		`INSERT INTO holdings (portfolio_id, symbol, shares, avg_cost_basis)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, portfolio_id, symbol, shares, avg_cost_basis, created_at`,
		portfolioID, req.Symbol, req.Shares, req.AvgCostBasis,
	).Scan(&h.ID, &h.PortfolioID, &h.Symbol, &h.Shares, &h.AvgCostBasis, &h.CreatedAt)
	if err != nil {
		log.Error().Err(err).Msg("Failed to add holding")
		writeError(w, http.StatusInternalServerError, "Failed to add holding")
		return
	}
	writeJSON(w, http.StatusCreated, h)
}

func (a *App) removeHolding(w http.ResponseWriter, r *http.Request) {
	holdingID := chi.URLParam(r, "holdingId")
	portfolioID := chi.URLParam(r, "id")
	_, err := a.db.Exec(r.Context(),
		`DELETE FROM holdings WHERE id = $1 AND portfolio_id = $2`, holdingID, portfolioID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to remove holding")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) getQuote(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")
	price, err := fetchStockPrice(a.config.AlphaVantageKey, symbol)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Failed to fetch quote: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, StockQuote{Symbol: symbol, Price: price})
}

func (a *App) swaggerDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<!DOCTYPE html>
<html><head><title>Portfolio Tracker API</title>
<style>
  body { font-family: system-ui, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #0f172a; color: #e2e8f0; }
  h1 { color: #38bdf8; } h2 { color: #7dd3fc; margin-top: 2em; }
  code { background: #1e293b; padding: 2px 6px; border-radius: 4px; font-size: 0.9em; }
  pre { background: #1e293b; padding: 16px; border-radius: 8px; overflow-x: auto; }
  .method { font-weight: bold; display: inline-block; width: 70px; }
  .get { color: #4ade80; } .post { color: #facc15; } .delete { color: #f87171; }
  a { color: #38bdf8; }
</style></head><body>
<h1>📈 Portfolio Tracker API</h1>
<p>Investment portfolio management with real-time stock prices, P&amp;L tracking, and allocation analysis.</p>

<h2>Endpoints</h2>
<pre>
<span class="method get">GET</span>    /api/v1/portfolios              List all portfolios
<span class="method post">POST</span>   /api/v1/portfolios              Create portfolio  {"name": "...", "description": "..."}
<span class="method get">GET</span>    /api/v1/portfolios/:id          Get portfolio with holdings + P&amp;L
<span class="method delete">DELETE</span> /api/v1/portfolios/:id          Delete portfolio

<span class="method post">POST</span>   /api/v1/portfolios/:id/holdings         Add holding  {"symbol": "AAPL", "shares": 10, "avg_cost_basis": 150.00}
<span class="method delete">DELETE</span> /api/v1/portfolios/:id/holdings/:hid    Remove holding

<span class="method get">GET</span>    /api/v1/quote/:symbol           Get current stock price

<span class="method get">GET</span>    /healthz                        Liveness probe
<span class="method get">GET</span>    /readyz                         Readiness probe
</pre>

<h2>Example</h2>
<pre>
# Create a portfolio
curl -X POST http://localhost:8080/api/v1/portfolios \
  -H "Content-Type: application/json" \
  -d '{"name": "Tech Growth", "description": "High-growth tech stocks"}'

# Add a holding
curl -X POST http://localhost:8080/api/v1/portfolios/1/holdings \
  -H "Content-Type: application/json" \
  -d '{"symbol": "AAPL", "shares": 50, "avg_cost_basis": 175.00}'

# View portfolio with P&amp;L
curl http://localhost:8080/api/v1/portfolios/1
</pre>

<h2>Response: Portfolio with P&amp;L</h2>
<pre>
{
  "id": 1,
  "name": "Tech Growth",
  "total_value": 11250.00,
  "total_cost": 8750.00,
  "total_pnl": 2500.00,
  "total_pnl_percent": 28.57,
  "holdings": [
    {
      "symbol": "AAPL",
      "shares": 50,
      "avg_cost_basis": 175.00,
      "current_price": 225.00,
      "market_value": 11250.00,
      "pnl": 2500.00,
      "pnl_percent": 28.57,
      "allocation_percent": 100.0
    }
  ]
}
</pre>
</body></html>`))
}

// --- Migrations ---

func runMigrations(cfg Config) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Migration: failed to connect")
	}
	defer pool.Close()

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS portfolios (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS holdings (
			id SERIAL PRIMARY KEY,
			portfolio_id INTEGER NOT NULL REFERENCES portfolios(id) ON DELETE CASCADE,
			symbol VARCHAR(20) NOT NULL,
			shares NUMERIC(15,4) NOT NULL,
			avg_cost_basis NUMERIC(15,4) NOT NULL DEFAULT 0,
			current_price NUMERIC(15,4),
			last_price_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_holdings_portfolio ON holdings(portfolio_id)`,
		`CREATE INDEX IF NOT EXISTS idx_holdings_symbol ON holdings(symbol)`,
	}

	for i, m := range migrations {
		if _, err := pool.Exec(ctx, m); err != nil {
			log.Fatal().Err(err).Int("step", i+1).Msg("Migration failed")
		}
		log.Info().Int("step", i+1).Msg("Migration applied")
	}
	log.Info().Msg("All migrations completed")
}

// --- Price Refresh ---

func runPriceRefresh(cfg Config) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Price refresh: failed to connect")
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `SELECT DISTINCT symbol FROM holdings`)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fetch symbols")
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		symbols = append(symbols, s)
	}

	log.Info().Int("count", len(symbols)).Msg("Refreshing prices")

	for _, symbol := range symbols {
		price, err := fetchStockPrice(cfg.AlphaVantageKey, symbol)
		if err != nil {
			log.Error().Err(err).Str("symbol", symbol).Msg("Failed to fetch price")
			continue
		}

		_, err = pool.Exec(ctx,
			`UPDATE holdings SET current_price = $1, last_price_at = NOW() WHERE symbol = $2`,
			price, symbol)
		if err != nil {
			log.Error().Err(err).Str("symbol", symbol).Msg("Failed to update price")
			continue
		}
		log.Info().Str("symbol", symbol).Float64("price", price).Msg("Price updated")

		// Alpha Vantage free tier: 5 calls/minute
		time.Sleep(15 * time.Second)
	}

	log.Info().Msg("Price refresh complete")
}

// --- Alpha Vantage ---

func fetchStockPrice(apiKey, symbol string) (float64, error) {
	url := fmt.Sprintf(
		"https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=%s",
		symbol, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		GlobalQuote struct {
			Price string `json:"05. price"`
		} `json:"Global Quote"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	if result.GlobalQuote.Price == "" {
		return 0, fmt.Errorf("no price data for %s (API limit may be reached)", symbol)
	}

	var price float64
	fmt.Sscanf(result.GlobalQuote.Price, "%f", &price)
	return price, nil
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
