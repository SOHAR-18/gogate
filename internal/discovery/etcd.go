package discovery

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	ServicePrefix = "services/"
	DialTimeout   = 5 * time.Second
	LeaseTTL      = 15
)

type Client struct {
	etcd *clientv3.Client
}

func NewClient(endpoints []string) (*Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: DialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}
	log.Printf("[DISCOVERY] Connected to etcd: %v", endpoints)
	return &Client{etcd: cli}, nil
}

func (c *Client) Register(ctx context.Context, serviceName, instanceURL string) error {
	key := ServicePrefix + serviceName + "/" + sanitize(instanceURL)

	lease, err := c.etcd.Grant(ctx, LeaseTTL)
	if err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	_, err = c.etcd.Put(ctx, key, instanceURL, clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	ch, err := c.etcd.KeepAlive(ctx, lease.ID)
	if err != nil {
		return fmt.Errorf("failed to keepalive: %w", err)
	}

	go func() {
		for range ch {
		}
		log.Printf("[DISCOVERY] Lease expired for %s", serviceName)
	}()

	log.Printf("[DISCOVERY] Registered: %s = %s", key, instanceURL)
	return nil
}

func (c *Client) GetServices(ctx context.Context, serviceName string) ([]string, error) {
	prefix := ServicePrefix + serviceName + "/"
	resp, err := c.etcd.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	urls := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		urls = append(urls, string(kv.Value))
	}
	return urls, nil
}

func (c *Client) Watch(ctx context.Context, serviceName string, onChange func([]string)) {
	prefix := ServicePrefix + serviceName + "/"
	log.Printf("[DISCOVERY] Watching: %s", prefix)

	go func() {
		watchCh := c.etcd.Watch(ctx, prefix, clientv3.WithPrefix())
		for range watchCh {
			urls, err := c.GetServices(ctx, serviceName)
			if err != nil {
				log.Printf("[DISCOVERY] Watch error: %v", err)
				continue
			}
			log.Printf("[DISCOVERY] Service %s updated: %v", serviceName, urls)
			onChange(urls)
		}
	}()
}

func (c *Client) Deregister(ctx context.Context, serviceName, instanceURL string) error {
	key := ServicePrefix + serviceName + "/" + sanitize(instanceURL)
	_, err := c.etcd.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to deregister: %w", err)
	}
	log.Printf("[DISCOVERY] Deregistered: %s", key)
	return nil
}

func (c *Client) Close() error {
	return c.etcd.Close()
}

func sanitize(url string) string {
	result := ""
	for _, ch := range url {
		if ch == '/' || ch == ':' {
			result += "-"
		} else {
			result += string(ch)
		}
	}
	return result
}
