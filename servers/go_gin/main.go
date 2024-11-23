package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	os.Mkdir("uploads", os.ModePerm)

	router.POST("/upload/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		file, _ := c.FormFile("file")
		fileData, _ := file.Open()
		defer fileData.Close()

		out, err := os.Create("uploads/" + filename)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to save file")
			return
		}
		defer out.Close()

		io.Copy(out, fileData)
		c.String(http.StatusOK, "File uploaded successfully")
	})

	router.GET("/download/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		filePath := "uploads/" + filename
		if _, err := os.Stat(filePath); err == nil {
			c.File(filePath)
		} else {
			c.String(http.StatusNotFound, "File not found")
		}
	})

	fmt.Println("Starting Gin server on port 9002...")
	router.Run(":9002")
}
