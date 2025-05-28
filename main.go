package main

import (
	"SipServe/MyHandlers"
	"SipServe/SipHandlers"
	"context"
	"fmt"
	"github.com/ghettovoice/gosip"
	"github.com/ghettovoice/gosip/log"
	"github.com/ghettovoice/gosip/sip"
	"github.com/ghettovoice/gosip/transaction"
	"github.com/ghettovoice/gosip/transport"
	"github.com/redis/go-redis/v9"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func InitRedis(ctx context.Context) *redis.Client {

	redisClientOption := redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}

	redisClient := redis.NewClient(&redisClientOption)

	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}
	return redisClient
}

/*func GetUserFromRedis(user string) (map[string]string, error) {
	return redisClient.HGetAll(ctx, fmt.Sprintf("user:%s", user)).Result()
}*/

/*func (h *SIPHandler) handleInviteRequest(req sip.Request, logger log.Logger, transportLayer transport.Layer) {
	toHeader, ok := req.To()
	if !ok {
		logger.Warn("Missing To header in INVITE")
		return
	}

	calleeURI := toHeader.Address.User().String()
	logger.Infof("INVITE for callee: %s", calleeURI)

	// Look up the callee in Redis
	key := fmt.Sprintf("user:%s", calleeURI)
	userData, err := redisClient.HGetAll(ctx, key).Result()
	if err != nil || len(userData) == 0 {
		logger.Warnf("User %s not found in Redis", calleeURI)

		resp := sip.NewResponseFromRequest("", req, 404, "Not Found", "")
		transportLayer.Send(resp)
		return
	}

	// Reconstruct Contact address
	calleeIP := userData["ip"]
	calleePortStr := userData["port"]
	calleePort, _ := strconv.Atoi(calleePortStr)

	var portCalle = sip.Port(calleePort)

	logger.Infof("Forwarding INVITE to %s:%s", calleeIP, calleePort)

	// Modify Request-URI
	newReq := req.Clone().(sip.Request)
	newReq.SetDestination(&sip.SipUri{
		FUser: sip.String{Str: calleeURI},
		FHost: calleeIP,
		FPort: &portCalle,
	})

	// Forward INVITE to the registered contact
	err = transportLayer.Send(newReq)
	if err != nil {
		logger.Errorf("Failed to forward INVITE: %v", err)
		resp := sip.NewResponseFromRequest("", req, 500, "Server Error", "")
		transportLayer.Send(resp)
		return
	}

	logger.Infof("INVITE forwarded to %s", calleeIP)
}*/

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	logger := log.NewDefaultLogrusLogger().WithPrefix("SIP-Server")

	handlers := MyHandlers.AsyncHandlers{
		RedisClient: InitRedis(context.Background()),
		Logger:      logger,
		Ctx:         context.Background(),
	}

	transportFactory := func(ip net.IP, dnsResolver *net.Resolver, msgMapper sip.MessageMapper, logger log.Logger) transport.Layer {
		return transport.NewLayer(ip, dnsResolver, msgMapper, logger)
	}

	transactionFactory := func(tpl sip.Transport, logger log.Logger) transaction.Layer {
		return transaction.NewLayer(tpl, logger)
	}

	srvConf := gosip.ServerConfig{
		Host:       "10.10.243.64",
		Dns:        "",
		Extensions: nil,
		MsgMapper:  nil,
		UserAgent:  "SIPServer/1.0",
	}

	srv := gosip.NewServer(
		srvConf,
		transportFactory,
		transactionFactory,
		logger,
	)

	mySipHandler := &SipHandlers.SipHandler{
		Logger: logger,
		Client: handlers.RedisClient,
		Ctx:    handlers.Ctx,
	}

	if err := srv.OnRequest(sip.REGISTER, mySipHandler.HandleRegisterRequest); err != nil {
		logger.Errorf("Failed to register request handler: %v", err)
		return
	}

	if err := srv.Listen("udp", "10.10.243.64:5060", nil); err != nil {
		logger.Errorf("Failed to listen on UDP 5060: %v", err)
		return
	}

	logger.Info("SIP server running on UDP 10.10.243.64:5060")

	<-stop
	srv.Shutdown()
}
