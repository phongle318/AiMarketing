package config

import (
	"github.com/kelseyhightower/envconfig"
)

var Bot botConfig

type botConfig struct {
	Port              int    `default:"1304"`
	DbHost            string `default:"10.3.16.19"`
	DbPort            string `default:"3306"`
	DbUsername        string `default:"fptai"` //root
	DbPassword        string `default:"fptai"`      //123456a@
	DbNameFptAi       string `default:"fptai"`
	DbNameQna         string `default:"qnamaker"`
	// Time in minutes for dialog to be expired if there's no message from user
	// Dialog will be restart if the last message is more than <DialogExpiredTime> minutes old
	DialogExpiredTime string `split_words:"true" default:"5"`

	StartDateDefault  string `split_words:"true" default:"2018-04-14"`
	EndDateDefault    string `split_words:"true" default:"2018-05-14"`
	DateSystemDefault string `split_words:"true" default:"2006-01-02"`

}

func LoadFromEnv() error {
	if err := envconfig.Process("user", &Bot); err != nil {
		return err
	}
	return nil
}
