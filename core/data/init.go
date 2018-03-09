/*******************************************************************************
 * Copyright 2017 Dell Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *
 * @microservice: core-data-go library
 * @author: Ryan Comer, Dell
 * @version: 0.5.0
 *******************************************************************************/
package data

import (
	"fmt"
	"strings"

	"github.com/tsconn23/edgex-go/core/clients/metadataclients"
	"github.com/tsconn23/edgex-go/core/data/clients"
	"github.com/tsconn23/edgex-go/core/data/messaging"
	consulclient "github.com/tsconn23/edgex-go/support/consul-client"
	"github.com/tsconn23/edgex-go/support/logging-client"
)

// Global variables
var dbc clients.DBClient
var loggingClient logger.LoggingClient
var ep *messaging.EventPublisher
var mdc metadataclients.DeviceClient
var msc metadataclients.ServiceClient

func ConnectToConsul(conf ConfigurationStruct) error {
	var err error

	// Initialize service on Consul
	err = consulclient.ConsulInit(consulclient.ConsulConfig{
		ServiceName:    conf.Servicename,
		ServicePort:    conf.Serverport,
		ServiceAddress: conf.Serviceaddress,
		CheckAddress:   conf.Consulcheckaddress,
		CheckInterval:  conf.Checkinterval,
		ConsulAddress:  conf.Consulhost,
		ConsulPort:     conf.Consulport,
	})

	if err != nil {
		return fmt.Errorf("connection to Consul could not be made: %v", err.Error())
	} else {
		// Update configuration data from Consul
		if err := consulclient.CheckKeyValuePairs(&conf, conf.Servicename, strings.Split(conf.Consulprofilesactive, ";")); err != nil {
			return fmt.Errorf("error getting key/values from Consul: %v", err.Error())
		}
	}
	return nil
}

func Init(conf ConfigurationStruct, l logger.LoggingClient) error {
	loggingClient = l
	configuration = conf
	//TODO: The above two are set due to global scope throughout the package. How can this be eliminated / refactored?
	//TODO: I should not have to pass the LoggingClient in here. See Heartbeat method above and how it uses channel
	var err error
	
	// Create a database client
	dbc, err = clients.NewDBClient(clients.DBConfiguration{
		DbType:       clients.MONGO,
		Host:         conf.Datamongodbhost,
		Port:         conf.Datamongodbport,
		Timeout:      conf.DatamongodbsocketTimeout,
		DatabaseName: conf.Datamongodbdatabase,
		Username:     conf.Datamongodbusername,
		Password:     conf.Datamongodbpassword,
	})
	if err != nil {
		return fmt.Errorf("couldn't connect to database: %v", err.Error())
	}

	// Create metadata clients
	mdc = metadataclients.NewDeviceClient(conf.Metadbdeviceurl)
	msc = metadataclients.NewServiceClient(conf.Metadbdeviceserviceurl)

	// Create the event publisher
	ep = messaging.NewZeroMQPublisher(messaging.ZeroMQConfiguration{
		AddressPort: conf.Zeromqaddressport,
	})

	return nil
}
