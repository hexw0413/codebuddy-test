package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"csgo2-trading-bot/config"
	"csgo2-trading-bot/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	db          *gorm.DB
	redis       *redis.Client
	steamConfig config.SteamConfig
}

type SteamUser struct {
	SteamID     string `json:"steamid"`
	PersonaName string `json:"personaname"`
	Avatar      string `json:"avatar"`
	AvatarFull  string `json:"avatarfull"`
}

type JWTClaims struct {
	UserID  uint   `json:"user_id"`
	SteamID string `json:"steam_id"`
	jwt.RegisteredClaims
}

func NewService(db *gorm.DB, redis *redis.Client, cfg config.SteamConfig) *Service {
	return &Service{
		db:          db,
		redis:       redis,
		steamConfig: cfg,
	}
}

// GenerateSteamLoginURL 生成Steam登录URL
func (s *Service) GenerateSteamLoginURL() string {
	params := url.Values{}
	params.Set("openid.ns", "http://specs.openid.net/auth/2.0")
	params.Set("openid.mode", "checkid_setup")
	params.Set("openid.return_to", s.steamConfig.CallbackURL)
	params.Set("openid.realm", s.steamConfig.CallbackURL)
	params.Set("openid.identity", "http://specs.openid.net/auth/2.0/identifier_select")
	params.Set("openid.claimed_id", "http://specs.openid.net/auth/2.0/identifier_select")

	return fmt.Sprintf("%s?%s", s.steamConfig.LoginURL, params.Encode())
}

// VerifySteamLogin 验证Steam登录回调
func (s *Service) VerifySteamLogin(query url.Values) (*models.User, error) {
	// 验证OpenID响应
	steamID, err := s.validateOpenIDResponse(query)
	if err != nil {
		return nil, err
	}

	// 获取Steam用户信息
	steamUser, err := s.getSteamUserInfo(steamID)
	if err != nil {
		return nil, err
	}

	// 查找或创建用户
	var user models.User
	result := s.db.Where("steam_id = ?", steamID).First(&user)
	
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 创建新用户
			user = models.User{
				SteamID:   steamID,
				Username:  steamUser.PersonaName,
				Avatar:    steamUser.AvatarFull,
				LastLogin: time.Now(),
			}
			if err := s.db.Create(&user).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, result.Error
		}
	} else {
		// 更新现有用户
		user.Username = steamUser.PersonaName
		user.Avatar = steamUser.AvatarFull
		user.LastLogin = time.Now()
		s.db.Save(&user)
	}

	return &user, nil
}

// validateOpenIDResponse 验证OpenID响应
func (s *Service) validateOpenIDResponse(query url.Values) (string, error) {
	// 构建验证请求
	params := url.Values{}
	params.Set("openid.assoc_handle", query.Get("openid.assoc_handle"))
	params.Set("openid.signed", query.Get("openid.signed"))
	params.Set("openid.sig", query.Get("openid.sig"))
	params.Set("openid.ns", query.Get("openid.ns"))
	params.Set("openid.mode", "check_authentication")

	signed := strings.Split(query.Get("openid.signed"), ",")
	for _, field := range signed {
		params.Set("openid."+field, query.Get("openid."+field))
	}

	// 发送验证请求到Steam
	resp, err := http.PostForm("https://steamcommunity.com/openid/login", params)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 解析响应
	var body []byte
	body = make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	response := string(body[:n])

	if !strings.Contains(response, "is_valid:true") {
		return "", errors.New("invalid steam login")
	}

	// 提取SteamID
	claimedID := query.Get("openid.claimed_id")
	parts := strings.Split(claimedID, "/")
	if len(parts) == 0 {
		return "", errors.New("invalid steam id")
	}

	return parts[len(parts)-1], nil
}

// getSteamUserInfo 获取Steam用户信息
func (s *Service) getSteamUserInfo(steamID string) (*SteamUser, error) {
	url := fmt.Sprintf("http://api.steampowered.com/ISteamUser/GetPlayerSummaries/v0002/?key=%s&steamids=%s",
		s.steamConfig.APIKey, steamID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Response struct {
			Players []SteamUser `json:"players"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Response.Players) == 0 {
		return nil, errors.New("steam user not found")
	}

	return &result.Response.Players[0], nil
}

// GenerateJWT 生成JWT令牌
func (s *Service) GenerateJWT(user *models.User) (string, error) {
	claims := JWTClaims{
		UserID:  user.ID,
		SteamID: user.SteamID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.steamConfig.SharedSecret))
}

// ValidateJWT 验证JWT令牌
func (s *Service) ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.steamConfig.SharedSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GenerateTOTP 生成TOTP令牌（用于Steam移动验证）
func (s *Service) GenerateTOTP(sharedSecret string) (string, error) {
	if sharedSecret == "" {
		return "", errors.New("shared secret is empty")
	}

	// 解码base32编码的密钥
	key, err := base32.StdEncoding.DecodeString(strings.ToUpper(sharedSecret))
	if err != nil {
		return "", err
	}

	// 获取当前时间戳（30秒为一个周期）
	counter := time.Now().Unix() / 30

	// 将计数器转换为字节数组
	buf := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		buf[i] = byte(counter)
		counter >>= 8
	}

	// 使用HMAC-SHA1生成哈希
	h := hmac.New(sha1.New, key)
	h.Write(buf)
	hash := h.Sum(nil)

	// 动态截取
	offset := hash[len(hash)-1] & 0xf
	code := int32(hash[offset]&0x7f)<<24 |
		int32(hash[offset+1]&0xff)<<16 |
		int32(hash[offset+2]&0xff)<<8 |
		int32(hash[offset+3]&0xff)

	// 生成5位数字代码
	code = code % 100000

	return fmt.Sprintf("%05d", code), nil
}

// SetupTwoFactor 设置双因素认证
func (s *Service) SetupTwoFactor(userID uint, sharedSecret, identitySecret string) error {
	// 加密存储密钥
	hashedShared, err := bcrypt.GenerateFromPassword([]byte(sharedSecret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	hashedIdentity, err := bcrypt.GenerateFromPassword([]byte(identitySecret), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"shared_secret":   string(hashedShared),
		"identity_secret": string(hashedIdentity),
	}).Error
}

// GetUserByID 根据ID获取用户
func (s *Service) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateTradeURL 更新交易链接
func (s *Service) UpdateTradeURL(userID uint, tradeURL string) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("trade_url", tradeURL).Error
}