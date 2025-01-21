package postgresqlcontroller

import (
	"context"
	"testing"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	"github.com/stretchr/testify/assert"
)

func Test_connectionDetails(t *testing.T) {
	tests := map[string]struct {
		givenUri         string
		expectedUser     string
		expectedPassword string
		expectedUri      string
		expectedHost     string
		expectedPort     string
		expectedDatabase string
	}{
		"FullURL": {
			givenUri:         "postgres://avnadmin:SUPERSECRET@instance-name-UUID.aivencloud.com:21699/defaultdb?sslmode=require",
			expectedUser:     "avnadmin",
			expectedPassword: "SUPERSECRET",
			expectedUri:      "postgres://avnadmin:SUPERSECRET@instance-name-UUID.aivencloud.com:21699/defaultdb?sslmode=require",
			expectedHost:     "instance-name-UUID.aivencloud.com",
			expectedPort:     "21699",
			expectedDatabase: "defaultdb",
		},
	}
	ctx := context.TODO()
	client := exoscalesdk.Client{}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			exo := exoscalesdk.DBAASServicePG{URI: tc.givenUri}
			secrets, err := connectionDetails(ctx, &exo, "somebase64string", &client)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedUser, string(secrets["POSTGRESQL_USER"]), "username")
			assert.Equal(t, tc.expectedPassword, string(secrets["POSTGRESQL_PASSWORD"]), "password")
			assert.Equal(t, tc.expectedUri, string(secrets["POSTGRESQL_URL"]), "full url")
			assert.Equal(t, tc.expectedHost, string(secrets["POSTGRESQL_HOST"]), "host name")
			assert.Equal(t, tc.expectedPort, string(secrets["POSTGRESQL_PORT"]), "port number")
			assert.Equal(t, tc.expectedDatabase, string(secrets["POSTGRESQL_DB"]), "database")
			assert.Equal(t, "somebase64string", string(secrets["ca.crt"]), "ca certificate")
		})
	}
}
