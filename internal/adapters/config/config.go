package config

type ServerConfig struct {
	Port int
}

type Config struct {
	Server ServerConfig
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 9090,
		},
	}
}
