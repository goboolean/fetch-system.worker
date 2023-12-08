package polygon_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Goboolean/common/pkg/resolver"
	"github.com/Goboolean/fetch-system.worker/internal/infrastructure/polygon"
	"github.com/stretchr/testify/assert"

	_ "github.com/Goboolean/common/pkg/env"
)



func SetupCryptoClient() *polygon.CryptoClient {
	c, err := polygon.NewCryptoClient(&resolver.ConfigMap{
		"SECRET_KEY": os.Getenv("POLYGON_SECRET_KEY"),
		"BUFFER_SIZE": 100000,
	})
	if err != nil {
		panic(err)
	}

	return c
}

func TeardownCryptoClient(c *polygon.CryptoClient) {
	fmt.Println("Closing")
	c.Close()
	fmt.Println("Closed")
}


func TestCryptoClient(t *testing.T) {

	t.Skip("Skipping the test, since it does not have privileges to access crypto api")

	c := SetupCryptoClient()
	defer TeardownCryptoClient(c)

	t.Run("Ping()", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 2)
		defer cancel()

		err := c.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("Subscribe()", func(t *testing.T) {
		ch, err := c.Subscribe()
		assert.NoError(t, err)

		time.Sleep(time.Second * 1)

		fmt.Println(len(ch))
	})
	
}