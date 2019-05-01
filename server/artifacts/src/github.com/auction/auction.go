/**
 * Copyright 2018 IT People Corporation. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the 'License');
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an 'AS IS' BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 * Author: Mohan Venkataraman <mohan.venkataraman@chainyard.com>
 * Author: Sandeep Pulluru <sandeep.pulluru@chainyard.com>
 * Author: Ratnakar Asara <ratnakar.asara@chainyard.com>
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = shim.NewLogger("example_cc0")

var isInit = false

// AuctionChaincode - Chaincode implementation
// ============================================
type AuctionChaincode struct {
	funcMap map[string]InvokeFunc
}

type InvokeFunc func(stub shim.ChaincodeStubInterface, args []string) pb.Response

// Constant for All function name that will be called from invoke
// ==============================================================
const (
	SAVE_IMAGE     string = "saveImage"
	GET_ITEM_BY_ID string = "getImageByID"
)

/////////////////////////////////////////////////////
const (
	IMAGE string = "IMAGE"
)

// Item - Record of the Asset (Inventory)
// Includes Description, title, certificate of authenticity or image whatever..idea is to checkin a image and store it
// in encrypted form
type Item struct {
	ItemID         string `json:"itemID,required"`
	DocType        string `json:"docType"`
	ItemDesc       string `json:"itemDescription"`
	ItemDetail     string `json:"itemDetail"` // Could included details such as who created the Art work if item is a Painting
	ItemDate       string `json:"itemDate"`
	ItemType       string `json:"itemType"`
	ItemSubject    string `json:"itemSubject"`
	ItemMedia      string `json:"itemMedia"`
	ItemSize       string `json:"itemSize"`
	ItemStatus     string `json:"itemStatus"`
	ItemImage      string `json:"itemImage"`
	ItemImageName  string `json:"itemImageName"`           // Item Subject + Extension
	AESKey         string `json:"aesKey"`                  // This is generated by the AES Algorithms
	ItemImageType  string `json:"itemImageType"`           // should be used to regenerate the appropriate image type
	ItemBasePrice  string `json:"itemBasePrice"`           // Reserve Price at Auction must be greater than this price
	CurrentOwnerID string `json:"currentOwnerID,required"` // This is validated for a user registered record
	TimeStamp      string `json:"timeStamp"`               // This is the time stamp
}

// initMaps - Map all the Functions here for Invoke
// ================================================
func (t *AuctionChaincode) initMaps() {
	t.funcMap = make(map[string]InvokeFunc)
	t.funcMap[SAVE_IMAGE] = saveImage
	t.funcMap[GET_ITEM_BY_ID] = getImageByID
}

// Init - Initialize Chaincode at Deploy Time
// ==========================================
func (t *AuctionChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Info("########### Init ###########")
	t.initMaps()
	isInit = true
	logger.SetLevel(shim.LogDebug)
	logger.Debug("AuctionChaincode Init")
	return getSuccessResponse("Succesfully Initiated Auction Chaincode")
}

// Invoke -Invoke Chaincode functions as requested by the Invoke Function
// ======================================================================
func (t *AuctionChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	//Temporay fix  if the initialization not done on the specific peer do it before Invoke a method
	if !isInit {
		t.initMaps()
		isInit = true
	}
	function, args := stub.GetFunctionAndParameters()
	logger.Infof("\n########### Invoke %s ###########\n", function)
	f, ok := t.funcMap[function]
	if ok {
		return f(stub, args)
	}

	logger.Errorf("Invalid function name %s", function)
	return getErrorResponse(fmt.Sprintf("Invalid function %s", function))
}

// getImageByID - Get Item Details by Item ID
// ==========================================
func getImageByID(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	logger.Debugf("Arguments for getImageByID : %s", args[0])

	// Get the Auction Item Information
	auctionImageBytes, err := queryObject(stub, IMAGE, []string{args[0]})
	if err != nil {
		return getErrorResponse("Failed to query if Auction Item exists")
	}
	if auctionImageBytes == nil {
		return getErrorResponse("Auction Item does not exist")
	}
	return shim.Success(auctionImageBytes)
}

// saveImage - creates a record of the Asset, store into chaincode state
// =====================================================================
func saveImage(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	logger.Debug("Arguments for saveImage : %s", args[0])

	if len(args) != 1 {
		return getErrorResponse("Incorrect number of arguments specified. Expecting 1")
	}

	auctionImage := &Item{}
	err := jsonToObject([]byte(args[0]), auctionImage)

	if err != nil {
		return getErrorResponse("Failed to convert arguments to a Image object")
	}

	auctionImageBytes, err := queryObject(stub, IMAGE, []string{auctionImage.ItemID})
	if err != nil {
		return getErrorResponse("Failed to query Image")
	}
	if auctionImageBytes != nil {
		return getErrorResponse("Image already exists")
	}

	auctionImage.ItemStatus = "INITIAL"
	auctionImageBytes, err = objectToJSON(auctionImage)
	err = updateObject(stub, IMAGE, []string{auctionImage.ItemID}, auctionImageBytes)
	if err != nil {
		return getErrorResponse("Unable to create Auction Item")
	}

	return shim.Success(auctionImageBytes)
}

// queryObject - Query a User Object by Object Name and Key
// This has to be a full key and should return only one unique object
// ==================================================================
func queryObject(stub shim.ChaincodeStubInterface, objectType string, keys []string) ([]byte, error) {
	// Check number of keys
	err := verifyAtLeastOneKeyIsPresent(keys)
	if err != nil {
		return nil, err
	}

	compoundKey, _ := stub.CreateCompositeKey(objectType, keys)
	logger.Debugf("queryObject() : Compound Key : %s", compoundKey)

	objBytes, err := stub.GetState(compoundKey)
	if err != nil {
		return nil, err
	}

	return objBytes, nil
}

// updateObject - Replace current data with replacement
// ====================================================
func updateObject(stub shim.ChaincodeStubInterface, objectType string, keys []string, objectData []byte) error {
	// Check number of keys
	err := verifyAtLeastOneKeyIsPresent(keys)
	if err != nil {
		return err
	}

	// Convert keys to  compound key
	compositeKey, _ := stub.CreateCompositeKey(objectType, keys)

	// Add Object JSON to state
	err = stub.PutState(compositeKey, objectData)
	if err != nil {
		logger.Errorf("updateObject() : Error inserting Object into State Database %s", err)
		return err
	}
	logger.Debugf("updateObject() : Successfully updated record of type %s", objectType)

	return nil
}

func main() {
	logger.SetLevel(shim.LogDebug)
	logger.Info("Auction: main(): Init ")
	err := shim.Start(new(AuctionChaincode))
	if err != nil {
		logger.Errorf("Auction: main(): Error starting Auction chaincode: %s", err)
	}
}

func init() {
	logger.Info("Called Init Method,Changing log Level : ", shim.LogDebug)
	logger.SetLevel(shim.LogDebug)
}

// Response -  Object to store Response Status and Message
// =======================================================
type Response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// getSuccessResponse - Create Success Response and return back to the calling application
// =======================================================================================
func getSuccessResponse(message string) pb.Response {
	objResponse := Response{Status: "200", Message: message}
	logger.Info("getSuccessResponse: Called For: ", objResponse)
	response, err := json.Marshal(objResponse)
	if err != nil {
		logger.Errorf(fmt.Sprintf("Invalid function %s", err))
	}
	return shim.Success(response)
}

// getErrorResponse - Create Error Response and return back to the calling application
// ===================================================================================
func getErrorResponse(message string) pb.Response {
	objResponse := Response{Status: "500", Message: message}
	logger.Info("getErrorResponse: Called For: ", objResponse)
	response, err := json.Marshal(objResponse)
	if err != nil {
		logger.Errorf(fmt.Sprintf("Invalid function %s", err))
	}
	return shim.Success(response)
}

///////// Utility Methods /////////

// jsonToObject (Serialize) : Unmarshalls a JSON into an object
// ============================================================
func jsonToObject(data []byte, object interface{}) error {
	if err := json.Unmarshal([]byte(data), object); err != nil {
		logger.Errorf("Unmarshal failed : %s ", err.Error()) //SCOMCONV004E
		return err
	}
	return nil
}

// objectToJSON (Deserialize) :  Marshalls an object into a JSON
// =============================================================
func objectToJSON(object interface{}) ([]byte, error) {
	var byteArray []byte
	var err error

	if byteArray, err = json.Marshal(object); err != nil {
		logger.Errorf("Marshal failed : %s ", err.Error())
		return nil, err
	}

	if len(byteArray) == 0 {
		return nil, fmt.Errorf(("failed to convert object"))
	}
	return byteArray, nil
}

// verifyAtLeastOneKeyIsPresent - This function verifies if the number of key
// provided is at least 1 and
// < the max keys defined for the Object
// ===========================================================================
func verifyAtLeastOneKeyIsPresent(args []string) error {
	// Check number of keys
	nKeys := len(args)

	if nKeys < 1 {
		err := fmt.Sprintf("verifyAtLeastOneKeyIsPresent() Failed: Atleast 1 Key must be needed :  nKeys : %s", strconv.Itoa(nKeys))
		logger.Debugf("verifyAtLeastOneKeyIsPresent() Failed: Atleast 1 Key must be needed :  nKeys : %s", strconv.Itoa(nKeys))
		return errors.New(err)
	}

	return nil
}
