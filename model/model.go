package model

import (
	"context"
	"strconv"
	"time"

	l "github.com/crosstyan/mqtt-to-ws/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Ctx    = context.TODO()
	logger = l.Lsugar
)

const (
	recordPerPage = 10
)

type MQTTMsg struct {
	Topic   string `json:"topic" example:"temperature"`
	Payload string `json:"payload" example:"23.5"`
}

func (m *MQTTMsg) ToRecord() (MQTTRecord, error) {
	payload, err := strconv.ParseFloat(m.Payload, 32)
	return MQTTRecord{
		Payload:   payload,
		Timestamp: time.Now(),
	}, err
}

type MQTTRecord struct {
	Payload float64 `bson:"payload" json:"payload" example:"24.23"`
	// Time RFC3339
	Timestamp time.Time `bson:"timestamp" json:"timestamp" example:"2020-01-01T00:00:00Z"`
}

func GetDB(uri string, db string) (*mongo.Database, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(Ctx, clientOptions)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	err = client.Ping(Ctx, nil)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	return client.Database(db), err
}

func CreateRecord(db *mongo.Database, collection string, data MQTTRecord) error {
	_, err := db.Collection(collection).InsertOne(Ctx, data)
	if err != nil {
		logger.Error(err)
	}

	return err
}

func GetRecords(db *mongo.Database, collection string, filter interface{}, opts *options.FindOptions) ([]MQTTRecord, error) {
	cur, err := db.Collection(collection).Find(Ctx, filter, opts)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	var results []MQTTRecord
	for cur.Next(Ctx) {
		var result MQTTRecord
		err := cur.Decode(&result)
		if err != nil {
			logger.Error(err)
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

func GetOptions(page int64, isDescend bool) *options.FindOptions {
	opts := options.Find()
	opts.SetLimit(recordPerPage)
	opts.SetSkip(recordPerPage * (page - 1))
	if isDescend {
		opts.SetSort(bson.M{"timestamp": -1})
	} else {
		opts.SetSort(bson.M{"timestamp": 1})
	}
	return opts
}

func GetRecordsByPage(db *mongo.Database, collection string, page int64) ([]MQTTRecord, error) {
	opts := GetOptions(page, true)
	// filter should not be nil
	return GetRecords(db, collection, bson.D{}, opts)
}

func GetRecordsFrom(db *mongo.Database, collection string, start time.Time, page int64, isDescend bool) ([]MQTTRecord, error) {
	opts := GetOptions(page, isDescend)
	// https://stackoverflow.com/questions/54548441/composite-literal-uses-unkeyed-fields
	filter := bson.D{
		{"timestamp", bson.D{
			{"$gte", start},
		}},
	}
	return GetRecords(db, collection, filter, opts)
}

func GetRecordsBetween(db *mongo.Database, collection string, start time.Time, end time.Time, page int64, isDescend bool) ([]MQTTRecord, error) {
	opts := GetOptions(page, isDescend)
	// https://stackoverflow.com/questions/54548441/composite-literal-uses-unkeyed-fields
	filter := bson.D{
		{"timestamp", bson.D{
			{"$gte", start},
			{"$lte", end},
		}},
	}
	return GetRecords(db, collection, filter, opts)
}

func HandleMQTTtoDB(mqttToDb chan MQTTMsg, db *mongo.Database) {
	for {
		msg := <-mqttToDb
		switch msg.Topic {
		case "temperature":
			val, err := msg.ToRecord()
			if err != nil {
				logger.Error(err)
				break
				// Prevent the execution of the following code
			}
			CreateRecord(db, "temperature", val)
		case "humidity":
			val, err := msg.ToRecord()
			if err != nil {
				logger.Error(err)
				break
			}
			CreateRecord(db, "humidity", val)
		default:
			// ignore
		}
	}
}
