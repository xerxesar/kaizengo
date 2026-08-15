package config

import "os"

func PostgresDSN() string {
	return os.Getenv("KaizenGo_POSTGRES_DSN")
}

func MongoURI() string {
	return os.Getenv("KaizenGo_MONGO_URI")
}
