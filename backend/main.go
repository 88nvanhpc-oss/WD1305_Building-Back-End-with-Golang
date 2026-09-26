package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type App struct {
	db        *sql.DB
	jwtSecret []byte
}

type User struct {
	ID         int64  `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	ProviderID string `json:"provider_id"`
}

type Transaction struct {
	ID          int64   `json:"id"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	CreatedAt   string  `json:"created_at"`
}

type transactionInput struct {
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Description string  `json:"description" binding:"required,min=1,max=200"`
	Type        string  `json:"type" binding:"required,oneof=income expense"`
}

func main() {
	_ = godotenv.Load()
	port := env("PORT", "8080")
	databaseURL := env("DATABASE_URL", "finance.db")
	secret := env("JWT_SECRET", "development-secret-change-me")

	db, err := sql.Open("sqlite", databaseURL)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err = initDB(db); err != nil {
		panic(err)
	}

	app := &App{db: db, jwtSecret: []byte(secret)}
	setupGoogle()

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: false,
	}))
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.POST("/register", app.register)
	r.POST("/login", app.login)
	r.GET("/auth/google", app.googleLogin)
	r.GET("/auth/google/callback", app.googleCallback)

	auth := r.Group("/")
	auth.Use(app.authMiddleware())
	auth.GET("/me", app.me)
	auth.POST("/transactions", app.createTransaction)
	auth.GET("/transactions", app.listTransactions)
	auth.DELETE("/transactions/:id", app.deleteTransaction)
	auth.GET("/reports", app.report)

	if err := r.Run(":" + port); err != nil {
		panic(err)
	}
}

func (a *App) me(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	var u User
	if err := a.db.QueryRow("SELECT id,email,COALESCE(name,''),provider,provider_id FROM users WHERE id=?", userID).Scan(&u.ID, &u.Email, &u.Name, &u.Provider, &u.ProviderID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}
	c.JSON(http.StatusOK, u)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 email TEXT NOT NULL UNIQUE,
 password_hash TEXT,
 name TEXT NOT NULL DEFAULT '',
 provider TEXT NOT NULL DEFAULT 'local',
 provider_id TEXT NOT NULL DEFAULT '',
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS transactions (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 user_id INTEGER NOT NULL,
 amount REAL NOT NULL CHECK(amount > 0),
 description TEXT NOT NULL,
 type TEXT NOT NULL CHECK(type IN ('income','expense')),
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
 FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_transactions_user ON transactions(user_id, created_at DESC);
`)
	return err
}

func (a *App) register(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email và mật khẩu không hợp lệ"})
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo tài khoản"})
		return
	}
	result, err := a.db.Exec("INSERT INTO users(email,password_hash,name) VALUES(?,?,?)", input.Email, string(hash), strings.TrimSpace(input.Name))
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			c.JSON(http.StatusConflict, gin.H{"error": "Email đã tồn tại"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo tài khoản"})
		return
	}
	id, _ := result.LastInsertId()
	token, _ := a.issueToken(id, input.Email)
	c.JSON(http.StatusCreated, gin.H{"token": token, "user": User{ID: id, Email: input.Email, Name: input.Name, Provider: "local"}})
}

func (a *App) login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email và mật khẩu không hợp lệ"})
		return
	}
	var u User
	var hash string
	err := a.db.QueryRow("SELECT id,email,COALESCE(name,''),provider,provider_id,COALESCE(password_hash,'') FROM users WHERE email=?", strings.ToLower(strings.TrimSpace(input.Email))).Scan(&u.ID, &u.Email, &u.Name, &u.Provider, &u.ProviderID, &hash)
	if err != nil || hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email hoặc mật khẩu không đúng"})
		return
	}
	token, _ := a.issueToken(u.ID, u.Email)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": u})
}

func (a *App) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Thiếu token xác thực"})
			return
		}
		token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("phương thức ký không hợp lệ")
			}
			return a.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ hoặc đã hết hạn"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
			return
		}
		id, ok := claims["user_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ"})
			return
		}
		c.Set("userID", int64(id))
		c.Next()
	}
}

func (a *App) issueToken(id int64, email string) (string, error) {
	claims := jwt.MapClaims{"user_id": id, "email": email, "exp": time.Now().Add(24 * time.Hour).Unix(), "iat": time.Now().Unix()}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.jwtSecret)
}

func (a *App) createTransaction(c *gin.Context) {
	var input transactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu giao dịch không hợp lệ"})
		return
	}
	userID := c.MustGet("userID").(int64)
	result, err := a.db.Exec("INSERT INTO transactions(user_id,amount,description,type) VALUES(?,?,?,?)", userID, input.Amount, strings.TrimSpace(input.Description), input.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu giao dịch"})
		return
	}
	id, _ := result.LastInsertId()
	var t Transaction
	err = a.db.QueryRow("SELECT id,amount,description,type,created_at FROM transactions WHERE id=? AND user_id=?", id, userID).Scan(&t.ID, &t.Amount, &t.Description, &t.Type, &t.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc giao dịch"})
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (a *App) listTransactions(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	rows, err := a.db.Query("SELECT id,amount,description,type,created_at FROM transactions WHERE user_id=? ORDER BY created_at DESC,id DESC", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải giao dịch"})
		return
	}
	defer rows.Close()
	items := make([]Transaction, 0)
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Amount, &t.Description, &t.Type, &t.CreatedAt); err == nil {
			items = append(items, t)
		}
	}
	c.JSON(http.StatusOK, items)
}

func (a *App) deleteTransaction(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID không hợp lệ"})
		return
	}
	result, err := a.db.Exec("DELETE FROM transactions WHERE id=? AND user_id=?", id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa giao dịch"})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy giao dịch"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa giao dịch"})
}

func (a *App) report(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	var income, expense float64
	err := a.db.QueryRow(`SELECT COALESCE(SUM(CASE WHEN type='income' THEN amount ELSE 0 END),0), COALESCE(SUM(CASE WHEN type='expense' THEN amount ELSE 0 END),0) FROM transactions WHERE user_id=?`, userID).Scan(&income, &expense)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo báo cáo"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"income": income, "expense": expense, "balance": income - expense})
}

func setupGoogle() {
	clientID, clientSecret := os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return
	}
	callback := env("GOOGLE_CALLBACK_URL", "http://localhost:8080/auth/google/callback")
	goth.UseProviders(google.New(clientID, clientSecret, callback, "email", "profile"))
}

func (a *App) googleLogin(c *gin.Context) {
	if os.Getenv("GOOGLE_CLIENT_ID") == "" || os.Getenv("GOOGLE_CLIENT_SECRET") == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google OAuth chưa được cấu hình"})
		return
	}
	if err := gothic.BeginAuthHandler(c.Writer, c.Request); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Không thể bắt đầu đăng nhập Google"})
	}
}

func (a *App) googleCallback(c *gin.Context) {
	if os.Getenv("GOOGLE_CLIENT_ID") == "" || os.Getenv("GOOGLE_CLIENT_SECRET") == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Google OAuth chưa được cấu hình"})
		return
	}
	providerUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Đăng nhập Google thất bại"})
		return
	}
	var u User
	err = a.db.QueryRow("SELECT id,email,COALESCE(name,''),provider,provider_id FROM users WHERE provider='google' AND provider_id=?", providerUser.UserID).Scan(&u.ID, &u.Email, &u.Name, &u.Provider, &u.ProviderID)
	if errors.Is(err, sql.ErrNoRows) {
		result, createErr := a.db.Exec("INSERT INTO users(email,name,provider,provider_id) VALUES(?,?,?,?)", providerUser.Email, providerUser.Name, "google", providerUser.UserID)
		if createErr != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Email Google đã tồn tại dưới dạng tài khoản thường"})
			return
		}
		u.ID, _ = result.LastInsertId()
		u.Email, u.Name, u.Provider, u.ProviderID = providerUser.Email, providerUser.Name, "google", providerUser.UserID
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc tài khoản Google"})
		return
	}
	token, _ := a.issueToken(u.ID, u.Email)
	frontend := env("FRONTEND_URL", "http://localhost:5500")
	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("%s/?token=%s", frontend, token))
}
