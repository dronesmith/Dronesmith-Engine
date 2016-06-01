package cloudlink

import (
  "os"
  "path"
  "bytes"
  "io/ioutil"
  "encoding/json"
  "path/filepath"
  "net/http"
  "fmt"
  "time"
  "config"
)

const (
  TICKER_INTERVAL = 15 // seconds
  MAX_UPLOAD_SIZE = 50000000
)

type FlightSyncer struct {
  FlightsPath string
  DroneId string
  UserId string
  isRunning bool

  lockname string
  quit chan bool
}

func NewFlightSyncer(fpath string) *FlightSyncer {
  return &FlightSyncer{
    fpath,
    "",
    "",
    false,
    "",
    make(chan bool),
  }
}

func (fs *FlightSyncer) IsRunning() bool {
  return fs.isRunning
}

func (fs *FlightSyncer) Lock(name string) {
  // config.Log(config.LOG_DEBUG, name)
  fs.lockname = path.Join(fs.FlightsPath, name)
}

func (fs *FlightSyncer) Unlock() {
  fs.lockname = ""
}

func (fs *FlightSyncer) Start(userId, droneId string) error {
  if userId == "" || droneId == "" || fs.isRunning {
    // config.Log(config.LOG_ERROR, "sy |", "User Id:", userId, "Drone Id:", droneId, "Running:", fs.isRunning)
    return fmt.Errorf("User Id and Drone Id required to start the syncer.")
  }

  fs.UserId = userId
  fs.DroneId = droneId

  go fs.listener()

  fs.isRunning = true

  return nil
}

func (fs *FlightSyncer) Stop() {
  fs.DroneId = ""
  fs.UserId = ""
  fs.isRunning = false
  fs.quit <- true
}

func (fs *FlightSyncer) listener() {

  checker := time.NewTimer(TICKER_INTERVAL * time.Second)

  for {
    select {
    case <-fs.quit:
      checker.Stop()
      return

    case <-checker.C:
      // check flight directory
      files, _ := filepath.Glob(fs.FlightsPath + "/Flight*")
      filesDone := make(chan bool, len(files))

      for _, f := range files {
        // verify the saver currently doesn't have the file
        if fs.lockname != f {
          // go fs.upload(f, filesDone)

          // The Edison doesn't have enough RAM to facilitate these in paralell,
          // so do this sequentially for now.
          fs.upload(f, filesDone)
        } else {
          // Can't sync it, we're done for now
          filesDone <- true
        }
      }

      // synchronize logic
      for _, _ = range files {
        c := <- filesDone
        if c == false {
          config.Log(config.LOG_ERROR, "sync: Error syncing a file")
        }
      }

      close(filesDone)

      // Note that we're manually resetting the timer here. This is because we
      // want to sync any new flights first before attempting another search.
      checker.Reset(TICKER_INTERVAL * time.Second)
    }
  }
}

func (fs *FlightSyncer) upload(fname string, done chan bool) {
  if file, err := os.OpenFile(path.Join(fname), os.O_RDWR, 0600); err != nil {
    done <- false
    return
  } else {
    // XXX - analyze memory footprint of this.
    chunk := make([]byte, MAX_UPLOAD_SIZE)

    if readBytes, err := file.Read(chunk); err != nil {
      done <- false
      return
    } else {
      buf := bytes.NewBuffer(chunk[:readBytes])

      // upload data
      res, err := http.Post("http://" + *config.DSCHttp + "/rt/mission/mavlinkBinary",
        "application/octet-stream", buf)
      if err != nil {
        config.Log(config.LOG_ERROR, err)
        done <- false
        return
      }

      body, err := ioutil.ReadAll(res.Body)
      if err != nil {
        config.Log(config.LOG_ERROR, err)
        done <- false
        return
      }

      res.Body.Close()
      file.Close()

      resMap := make(map[string]string)
      if err := json.Unmarshal(body, &resMap); err != nil {
        config.Log(config.LOG_ERROR, err)
        done <- false
        return
      } else if resMap["status"] == "OK" {
        // synthesize the JSON
        sendMap := make(map[string]string)
        sendMap["user"] = fs.UserId
        sendMap["drone"] = fs.DroneId

        if jsonChunk, err := json.Marshal(sendMap); err != nil {
          config.Log(config.LOG_ERROR, "Could not build JSON")
          done <- false
          return
        } else {

          // send associate request
          body, err := put("http://" + *config.DSCHttp + "/rt/mission/" + resMap["id"] + "/associate",
            "application/json", bytes.NewBuffer(jsonChunk))
          if err != nil {
            config.Log(config.LOG_ERROR, err)
            done <- false
            return
          } else {

            if err := json.Unmarshal(body, &resMap); err != nil {
              config.Log(config.LOG_ERROR, err)
              done <- false
              return
            } else {
              if resMap["status"] == "OK" {
                // remove the file
                if err := os.Remove(fname); err != nil {
                  config.Log(config.LOG_ERROR, "Could not remove file.")
                  done <- false
                  return
                }

                config.Log(config.LOG_INFO, "File successfully synced!")
                done <- true
                return
              } else {
                config.Log(config.LOG_ERROR, "Association failed.")
                done <- false
                return
              }
            }
          }
        }
      } else {
        config.Log(config.LOG_ERROR, "Upload failed: ", resMap["error"])
        done <- false
        return
      }
    }
  }
}

func put(url, ctype string, data *bytes.Buffer) ([]byte, error) {
	client := &http.Client{}
	request, err := http.NewRequest("PUT", url, data)
	request.ContentLength = int64(data.Len())
  request.Header.Add("Content-Type", ctype)
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
		  return nil, err
		}

    return contents, nil
	}
}
