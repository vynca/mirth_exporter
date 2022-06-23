package main

import (
	"fmt"
	"os"

)

func goDotEnvVariable(key string) string {
//    err := godotenv.Load(".env")
//
//    if err != nil {
//      log.Fatalf("Error loading .env file")
//    }

  return os.Getenv(key)
 }

// func check(e error) {
//     if e != nil {
//         panic(e)
//     }
// }
 func load_properties() {
// todo fix ?? if then , add to main package
   godotenv.Load(".env")
   defaultMirthServiceUrl := "https://localhost:8443"
   mirthServiceURL := goDotEnvVariable("MIRTH_SERVICE_URL") ?? defaultMirthServiceUrl
   defaultMirthUsernamePassword := "admin"
   mirthUsername := goDotEnvVariable("USERNAME") ?? defaultMirthUsernamePassword
   mirthPassword := goDotEnvVariable("PASSWORD") ?? defaultMirthUsernamePassword
   mirthVersion := "3.11.0"

  file := os.Create("mirth-cli-config.properties")
//       check(err)

    defer file.Close()
     file.WriteString(fmt.Sprintf("address=%s\n",mirthServiceURL))
     file.WriteString(fmt.Sprintf("user=%s\n",mirthUsername))
     file.WriteString(fmt.Sprintf("password=%s\n",mirthPassword))
     file.WriteString(fmt.Sprintf("version=%s\n",mirthVersion))
}
