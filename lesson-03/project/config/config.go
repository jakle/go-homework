package config

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Upload   UploadConfig   `mapstructure:"upload"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"` // mysql, postgres, sqlite
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"` // 仅PostgreSQL使用
	Charset  string `mapstructure:"charset"` // 仅MySQL使用
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire string `mapstructure:"expire"`
}
type UploadConfig struct {
	Path       string `mapstructure:"path"`
	MaxSize    int64  `mapstructure:"max_size"`
	AllowedExt string `mapstructure:"allowed_ext"`
	URLPrefix  string `mapstructure:"url_prefix"`
}

func Load() *Config {
	// 简化配置加载，实际应该使用 Viper
	return &Config{
		Server: ServerConfig{
			Port: "8080",
			Host: "0.0.0.0",
			Mode: "debug",
		},
		Database: DatabaseConfig{
			Driver:   "sqlite", // 默认使用sqlite
			Host:     "localhost",
			Port:     3306,
			Username: "root",
			Password: "password",
			DBName:   "mydb",
			SSLMode:  "disable",
			Charset:  "utf8mb4",
		},
		JWT: JWTConfig{
			Secret: "your-secret-key-change-in-production",
			Expire: "24h",
		},
		Upload: UploadConfig{
			Path:       "uploads",
			MaxSize:    5 << 20, //5MB
			AllowedExt: "jpg,jpeg,png,gif",
			URLPrefix:  "/uploads",
		},
	}
}
