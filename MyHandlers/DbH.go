package MyHandlers

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

func StoreOrUpdateUserInRedis(ctx context.Context, client *redis.Client, user string, ip string, port string) error {
	key := fmt.Sprintf("user:%s", user)

	data := map[string]interface{}{
		"ip":   ip,
		"port": port,
	}

	if err := client.HSet(ctx, key, data).Err(); err != nil {
		return err
	}

	exists, err := client.Exists(ctx, key).Result()
	if err != nil {
		return err
	}

	if exists == 1 {
		return client.Expire(ctx, key, 40*time.Second).Err()
	}

	return nil
}
