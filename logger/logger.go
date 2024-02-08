package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Save the health status to a file
func logHealthStatus(success bool) {
	status := "FAIL"
	if success {
		status = "OK"
	}
	os.WriteFile("/tmp/health_status", []byte(status), 0644)
}

// ExecuteCommand executes a shell command and returns its output
func executeCommand(command string) (string, error) {
	out, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		return "", fmt.Errorf("error executing command: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetUsageValues parses command output and returns usage values as a slice of float64
func getUsageValues(command string) ([]float64, error) {
	out, err := executeCommand(command)
	if err != nil {
		return nil, err
	}

	var results []float64
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		if value, err := strconv.ParseFloat(scanner.Text(), 64); err == nil {
			results = append(results, value)
		}
	}
	return results, nil
}

// The Raspberry Pi has a built-in temperature sensor that can be read from the filesystem
func getSysTemperature() (float64, error) {
	bytes, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return 0, err
	}

	tempStr := strings.TrimSpace(string(bytes))
	tempMilli, err := strconv.ParseInt(tempStr, 10, 64)
	if err != nil {
		return 0, err
	}

	// Convert from millidegrees to degrees Celsius
	return float64(tempMilli) / 1000.0, nil
}

// Grab an env variable with a known integer default value
func getEnvOrDefault(key string, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func main() {
	// Grab MongoDB config from environment variables
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI is not set")
	}
	databaseName := os.Getenv("MONGO_DATABASE")
	if databaseName == "" {
		log.Fatalf("MONGO_DATABASE not provided in the .env file.")
	}
	collectionName := os.Getenv("MONGO_LOGGER_COLLECTION")
	if collectionName == "" {
		log.Fatalf("MONGO_LOGGER_COLLECTION not provided in the .env file.")
	}

	// Grab logger config
	sleepDuration := getEnvOrDefault("SLEEP_DURATION_SECS", 60)
	numSamples := getEnvOrDefault("NUM_SAMPLES", 3)
	intervalSeconds := getEnvOrDefault("SAMPLE_INTERVAL_SECS", 20)

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	db := client.Database(databaseName)

	// Check if the collection exists
	collectionNames, err := db.ListCollectionNames(context.TODO(), bson.M{"name": collectionName})
	if err != nil {
		log.Fatal(err)
	}

	// If the collection does not exist, create it as a time series collection
	if len(collectionNames) == 0 {
		// Create a time series collection
		tso := options.TimeSeries().SetTimeField("timestamp")
		opts := options.CreateCollection().SetTimeSeriesOptions(tso)
		db.CreateCollection(context.TODO(), collectionName, opts)

		// Add an increasing index on the "timestamp" key, ascending order.
		indexModel := mongo.IndexModel{
			Keys: bson.M{"timestamp": 1},
		}
		_, err := db.Collection(collectionName).Indexes().CreateOne(context.TODO(), indexModel)
		if err != nil {
			log.Fatal(err)
		}
	}

	collection := db.Collection(collectionName)

	// Collect resource data and insert into MongoDB
	for {
		var cpuUsages, ramUsages, ioUsages, temps float64

		for i := 0; i < numSamples; i++ {
			cpuUsage, errCpu := getUsageValues("mpstat 1 1 | awk '/^Average/ {print 100 - $NF}'")
			ramUsage, errRam := getUsageValues("free | awk '/Mem:/ {printf \"%.2f\", $3/$2 * 100.0}'")
			ioUsage, errIo := getUsageValues("iostat -d 1 2 | awk '/Device/ {report++} report==2 && /^[^ ]/ && !/Device:/ {sum += $2} END{print sum}'")
			temp, errTemp := getSysTemperature()

			// Check for errors and use NaN if any readings fail
			if errCpu != nil {
				cpuUsages += math.NaN()
			} else {
				cpuUsages += cpuUsage[0]
			}
			if errRam != nil {
				ramUsages += math.NaN()
			} else {
				ramUsages += ramUsage[0]
			}
			if errIo != nil {
				ioUsages += math.NaN()
			} else {
				ioUsages += ioUsage[0]
			}
			if errTemp != nil {
				temps += math.NaN()
			} else {
				temps += temp
			}

			if i < numSamples-1 {
				time.Sleep(time.Duration(intervalSeconds) * time.Second)
			}
		}

		// Create the document
		floatNumSamples := float64(numSamples)
		doc := bson.M{
			"timestamp":   time.Now(),
			"cpu_total":   cpuUsages / floatNumSamples,
			"ram":         ramUsages / floatNumSamples,
			"io_requests": ioUsages / floatNumSamples,
			"temperature": temps / floatNumSamples,
		}

		// Insert the document into MongoDB and check for success
		success := false
		if _, err := collection.InsertOne(context.TODO(), doc); err != nil {
			log.Fatal("Failed to insert document: %v\n", err)
		} else {
			success = true
		}

		// Log the health status based on the success of the insert operation
		logHealthStatus(success)

		// Wait before the next iteration
		time.Sleep(time.Duration(sleepDuration) * time.Second)
	}
}
