package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	//fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get thumbnail from form data", err)
		return
	}

	defer file.Close()

	contentType := header.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", err)
		return
	}

	if mediaType != "image/jpg" && mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", fmt.Errorf("content type must be image/jpg, image/jpeg, or image/png"))
		return
	}

	// imgData, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Couldn't read thumbnail data", err)
	// 	return
	// }

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video data", err)
		return
	}

	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You don't have permission to upload a thumbnail for this video", fmt.Errorf("user %s doesn't have permission to upload thumbnail for video %s", userID, videoID))
		return
	}

	fileExtension := strings.Split(contentType, "/")[1]
	filePath := filepath.Join(cfg.assetsRoot, videoIDString+"."+fileExtension)

	newThumbnail, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail file", err)
		return
	}
	defer newThumbnail.Close()

	if _, err := io.Copy(newThumbnail, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't write thumbnail data to file", err)
		return
	}

	//base64Thumbnail := base64.StdEncoding.EncodeToString(imgData)
	// // newThumbnail := thumbnail{
	// // 	data:      imgData,
	// // 	mediaType: contentType,
	// // }

	////videoThumbnails[videoID] = newThumbnail

	////thumbnailURL := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videoIDString)
	//thumbnailURL := fmt.Sprintf("data:%s;base64,%s", contentType, base64Thumbnail)
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, videoIDString+"."+fileExtension)
	videoData.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video with thumbnail URL", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoData)
}
