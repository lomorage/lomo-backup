package main

import (
	"github.com/lomorage/lomo-backup/common/gcloud"
	"github.com/urfave/cli"
)

func gcloudAuth(ctx *cli.Context) error {
	return gcloud.AuthHelper(ctx.String("redirect-path"), ctx.Int("redirect-port"),
		&gcloud.Config{CredFilename: ctx.String("cred"), TokenFilename: ctx.String("token")})
}
