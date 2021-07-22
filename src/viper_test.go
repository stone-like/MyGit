package src

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type User struct {
	Name  string
	Email string
}

func TestReadConfig(t *testing.T) {
	v := viper.New()
	v.AddConfigPath("../")
	v.SetConfigName(".mygit")
	v.SetConfigType("yaml")

	err := v.ReadInConfig()

	assert.NoError(t, err)

	assert.Equal(t, "test", v.GetString("name"))
}
