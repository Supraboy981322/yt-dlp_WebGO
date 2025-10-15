package main

import (
    "fmt"
    "log"
    "log/slog"
    "net/http"
    "net"
    "os"
    "io/ioutil"
    "time"
    "encoding/json"
    "bytes"
    "context"
    "strconv"

//    "golang.org/x/net/websocket"

    "github.com/gorilla/websocket"
    "github.com/BurntSushi/toml"
    "github.com/lrstanley/go-ytdlp"
)

type (
    ServerSettings struct {
        Server string `toml:"server"`
        Port int `toml:"port"`
    }

    //struct for settings.toml
    Settings struct {
        Name string `toml:"name"`
        Server ServerSettings
    }
)

/**************
**  main fn  **
**************/
func main() {
    //install yt-dlp and cache it if not already installed
    ytdlp.MustInstall(context.TODO(), nil)

    port := strconv.Itoa(readSettings().Server.Port)
    fmt.Println("yt-dlp WebGO starting...")
    fmt.Printf("  name:  %s\n", readSettings().Name)
    
    http.HandleFunc("/save", saveHandler)
//    http.Handle("/progress", websocket.Handler(dlProgress))
    http.HandleFunc("/", webHandler)

    ipAddressArray, err := net.InterfaceAddrs()
    if err != nil {
        fmt.Errorf("err detecting ip address:  %v\n", err)
    }

    for _, ipAddress := range ipAddressArray {
        if ipNet, ok := ipAddress.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
            if ipNet.IP.To4() != nil {
                fmt.Printf("listening on http://%s:%s\n", ipNet.IP, port)
            }
        }
    }

    log.Fatal(http.ListenAndServe(":"+port, nil))
}


/******************************
** fn to read settings.json  **
******************************/
func readSettings() Settings {
    var settings Settings
    _, err := toml.DecodeFile("settings.toml", &settings)
    
    if err != nil {
        fmt.Errorf("err reading settings file:  %v\n", err)
    }

    return settings
}


/****************************
** fn to serve the web ui  **
****************************/
func webHandler(w http.ResponseWriter, r *http.Request) {
    requestedPage := fmt.Sprintf("web/%s", r.URL.Path[1:])

    fmt.Printf("requestedPage == %s\n", requestedPage)
    
    webPageContent, err := ioutil.ReadFile(requestedPage)
    if err != nil {
        fmt.Printf("err reading file for requested page:  %v\n", err)
        http.NotFound(w, r)
        return
    }

    webpageReader := bytes.NewReader(webPageContent)

    http.ServeContent(w, r, requestedPage, time.Now(), webpageReader)
}


/****************************************
 *  fn to handle dl progress websocket  *
 ****************************************/
func dlProgress(w http.ResponseWriter, r *http.Request) {
    progressJSON, err := ioutil.ReadFile("progress.json")
    if err != nil {
        fmt.Errorf("failed to read progress.json")
    }
    jsonReader := bytes.NewReader(progressJSON)

    go writer(ws, lastMod)
}


/********************************************
 *  fn to recieve requests to save a video  *
 ********************************************/
func saveHandler(w http.ResponseWriter, r *http.Request) {
    //get the `url` header
    url := r.Header.Get("url")
    //get the `format` header
    format := r.Header.Get("format")
    //log the request
    fmt.Printf("saveRequest {\n    \"url\": \"%s\",\n    \"format\": \"%s\"\n}\n", url, format)
    //send something, anything to the client
    w.Write([]byte("recieved"))
    w.Write([]byte("initializing dl..."))

    w.WriteHeader(http.StatusOK)

    //create the yt-dlp process
    dl := ytdlp.New().
        PrintJSON().
        NoProgress().
        FormatSort("res").
        RecodeVideo("mp4").
        Continue().
        ProgressFunc(100*time.Millisecond, func(prog ytdlp.ProgressUpdate) {
            data := []byte(fmt.Sprintf(
                "%s @ %s [etc: %s] :: %s\n",
                prog.Status,
                prog.PercentString(),
                prog.ETA(),
                prog.Filename,))
            err := os.WriteFile("progress.json", data, 0644)
            if err != nil {
                fmt.Errorf("err writing progress.json:  %v\n", err)
            }
        }).
        Output("%(title)s.%(ext)s")
    
    //download the video from the url
    result, err := dl.Run(context.Background(), url)
    if err != nil {
        log.Fatal(err)
    }
    err = json.NewEncoder(w).Encode(result)
    if err != nil {
        slog.ErrorContext(r.Context(), "failed to encode result", "error", err)
        return
    }
}


/******************************************************
**  fn to recieve download requests for saved video  **
******************************************************/
func dlHandler(w http.ResponseWriter, r *http.Request) {
    //get the `file` tag in the header
    requestedDL := r.Header.Get("file")
    //log the request
    fmt.Printf("dlRequest == %s\n", requestedDL)
    //return something, anything to the client
    w.Write([]byte("recieved"))
}
