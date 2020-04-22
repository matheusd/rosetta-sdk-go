// Copyright 2020 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package asserter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	var (
		validNetwork = &types.NetworkIdentifier{
			Blockchain: "hello",
			Network:    "world",
		}

		validNetworkStatus = &types.NetworkStatusResponse{
			GenesisBlockIdentifier: &types.BlockIdentifier{
				Index: 0,
				Hash:  "block 0",
			},
			CurrentBlockIdentifier: &types.BlockIdentifier{
				Index: 100,
				Hash:  "block 100",
			},
			CurrentBlockTimestamp: MinUnixEpoch + 1,
			Peers: []*types.Peer{
				{
					PeerID: "peer 1",
				},
			},
		}

		invalidNetworkStatus = &types.NetworkStatusResponse{
			CurrentBlockIdentifier: &types.BlockIdentifier{
				Index: 100,
				Hash:  "block 100",
			},
			CurrentBlockTimestamp: MinUnixEpoch + 1,
			Peers: []*types.Peer{
				{
					PeerID: "peer 1",
				},
			},
		}

		validNetworkOptions = &types.NetworkOptionsResponse{
			Version: &types.Version{
				RosettaVersion: "1.2.3",
				NodeVersion:    "1.0",
			},
			Allow: &types.Allow{
				OperationStatuses: []*types.OperationStatus{
					{
						Status:     "Success",
						Successful: true,
					},
				},
				OperationTypes: []string{
					"Transfer",
				},
				Errors: []*types.Error{
					{
						Code:      1,
						Message:   "error",
						Retriable: true,
					},
				},
			},
		}

		invalidNetworkOptions = &types.NetworkOptionsResponse{
			Version: &types.Version{
				RosettaVersion: "1.2.3",
				NodeVersion:    "1.0",
			},
			Allow: &types.Allow{
				OperationTypes: []string{
					"Transfer",
				},
				Errors: []*types.Error{
					{
						Code:      1,
						Message:   "error",
						Retriable: true,
					},
				},
			},
		}

		duplicateStatuses = &types.NetworkOptionsResponse{
			Version: &types.Version{
				RosettaVersion: "1.2.3",
				NodeVersion:    "1.0",
			},
			Allow: &types.Allow{
				OperationStatuses: []*types.OperationStatus{
					{
						Status:     "Success",
						Successful: true,
					},
					{
						Status:     "Success",
						Successful: false,
					},
				},
				OperationTypes: []string{
					"Transfer",
				},
				Errors: []*types.Error{
					{
						Code:      1,
						Message:   "error",
						Retriable: true,
					},
				},
			},
		}

		duplicateTypes = &types.NetworkOptionsResponse{
			Version: &types.Version{
				RosettaVersion: "1.2.3",
				NodeVersion:    "1.0",
			},
			Allow: &types.Allow{
				OperationStatuses: []*types.OperationStatus{
					{
						Status:     "Success",
						Successful: true,
					},
				},
				OperationTypes: []string{
					"Transfer",
					"Transfer",
				},
				Errors: []*types.Error{
					{
						Code:      1,
						Message:   "error",
						Retriable: true,
					},
				},
			},
		}
	)

	var tests = map[string]struct {
		network        *types.NetworkIdentifier
		networkStatus  *types.NetworkStatusResponse
		networkOptions *types.NetworkOptionsResponse

		err error
	}{
		"valid responses": {
			network:        validNetwork,
			networkStatus:  validNetworkStatus,
			networkOptions: validNetworkOptions,

			err: nil,
		},
		"invalid network status": {
			network:        validNetwork,
			networkStatus:  invalidNetworkStatus,
			networkOptions: validNetworkOptions,

			err: errors.New("BlockIdentifier is nil"),
		},
		"invalid network options": {
			network:        validNetwork,
			networkStatus:  validNetworkStatus,
			networkOptions: invalidNetworkOptions,

			err: errors.New("no Allow.OperationStatuses found"),
		},
		"duplicate operation statuses": {
			network:        validNetwork,
			networkStatus:  validNetworkStatus,
			networkOptions: duplicateStatuses,

			err: errors.New("Allow.OperationStatuses contains a duplicate Success"),
		},
		"duplicate operation types": {
			network:        validNetwork,
			networkStatus:  validNetworkStatus,
			networkOptions: duplicateTypes,

			err: errors.New("Allow.OperationTypes contains a duplicate Transfer"),
		},
	}

	for name, test := range tests {
		t.Run(fmt.Sprintf("%s with responses", name), func(t *testing.T) {
			asserter, err := NewClientWithResponses(
				test.network,
				test.networkStatus,
				test.networkOptions,
			)

			assert.Equal(t, test.err, err)

			if test.err != nil {
				return
			}

			assert.NotNil(t, asserter)
			network, genesis, opTypes, opStatuses, errors, err := asserter.ClientConfiguration()
			assert.NoError(t, err)
			assert.Equal(t, test.network, network)
			assert.Equal(t, test.networkStatus.GenesisBlockIdentifier, genesis)
			assert.ElementsMatch(t, test.networkOptions.Allow.OperationTypes, opTypes)
			assert.ElementsMatch(t, test.networkOptions.Allow.OperationStatuses, opStatuses)
			assert.ElementsMatch(t, test.networkOptions.Allow.Errors, errors)
		})

		t.Run(fmt.Sprintf("%s with file", name), func(t *testing.T) {
			fileConfig := FileConfiguration{
				NetworkIdentifier:        test.network,
				GenesisBlockIdentifier:   test.networkStatus.GenesisBlockIdentifier,
				AllowedOperationTypes:    test.networkOptions.Allow.OperationTypes,
				AllowedOperationStatuses: test.networkOptions.Allow.OperationStatuses,
				AllowedErrors:            test.networkOptions.Allow.Errors,
			}
			tmpfile, err := ioutil.TempFile("", "test.json")
			assert.NoError(t, err)
			defer os.Remove(tmpfile.Name())

			file, err := json.MarshalIndent(fileConfig, "", " ")
			assert.NoError(t, err)

			_, err = tmpfile.Write(file)
			assert.NoError(t, err)
			assert.NoError(t, tmpfile.Close())

			asserter, err := NewClientWithFile(
				tmpfile.Name(),
			)

			assert.Equal(t, test.err, err)

			if test.err != nil {
				return
			}

			assert.NotNil(t, asserter)
			network, genesis, opTypes, opStatuses, errors, err := asserter.ClientConfiguration()
			assert.NoError(t, err)
			assert.Equal(t, test.network, network)
			assert.Equal(t, test.networkStatus.GenesisBlockIdentifier, genesis)
			assert.ElementsMatch(t, test.networkOptions.Allow.OperationTypes, opTypes)
			assert.ElementsMatch(t, test.networkOptions.Allow.OperationStatuses, opStatuses)
			assert.ElementsMatch(t, test.networkOptions.Allow.Errors, errors)
		})
	}
}
