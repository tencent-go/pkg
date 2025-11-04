package api

import (
	"github.com/tencent-go/pkg/errx"
	"github.com/tencent-go/pkg/types"
)

type ContentType string

const (
	ContentTypeImageJpeg ContentType = "image/jpeg"
	ContentTypeImagePng  ContentType = "image/png"
	ContentTypeImageGif  ContentType = "image/gif"
	ContentTypeImageBmp  ContentType = "image/bmp"
	ContentTypeImageSvg  ContentType = "image/svg+xml"
	ContentTypeImageTiff ContentType = "image/tiff"

	ContentTypeAudioMpeg ContentType = "audio/mpeg"
	ContentTypeAudioWav  ContentType = "audio/wav"
	ContentTypeAudioOgg  ContentType = "audio/ogg"
	ContentTypeAudioFlac ContentType = "audio/flac"
	ContentTypeAudioAac  ContentType = "audio/aac"

	ContentTypeVideoMp4  ContentType = "video/mp4"
	ContentTypeVideoMpeg ContentType = "video/mpeg"
	ContentTypeVideoWebm ContentType = "video/webm"
	ContentTypeVideoAvi  ContentType = "video/x-msvideo"
	ContentTypeVideoMov  ContentType = "video/quicktime"

	ContentTypeTextPlain      ContentType = "text/plain"
	ContentTypeTextHtml       ContentType = "text/html"
	ContentTypeTextCss        ContentType = "text/css"
	ContentTypeTextJavascript ContentType = "text/javascript"
	ContentTypeTextMarkdown   ContentType = "text/markdown"
	ContentTypeTextCsv        ContentType = "text/csv"

	ContentTypeApplicationJson           ContentType = "application/json"
	ContentTypeApplicationXml            ContentType = "application/xml"
	ContentTypeApplicationPdf            ContentType = "application/pdf"
	ContentTypeApplicationZip            ContentType = "application/zip"
	ContentTypeApplicationGzip           ContentType = "application/gzip"
	ContentTypeApplicationExcelXlsx      ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	ContentTypeApplicationWordDocx       ContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	ContentTypeApplicationPowerpointPptx ContentType = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	ContentTypeApplicationFormUrlencoded ContentType = "application/x-www-form-urlencoded"
	ContentTypeApplicationOctetStream    ContentType = "application/octet-stream"
	ContentTypeApplicationRtf            ContentType = "application/rtf"
	ContentTypeApplicationJavascript     ContentType = "application/javascript"
)

func (c ContentType) Enum() types.Enum {
	return types.RegisterEnum(ContentTypeImageJpeg, ContentTypeImagePng, ContentTypeImageGif, ContentTypeImageBmp, ContentTypeImageSvg, ContentTypeImageTiff,
		ContentTypeAudioMpeg, ContentTypeAudioWav, ContentTypeAudioOgg, ContentTypeAudioFlac, ContentTypeAudioAac,
		ContentTypeVideoMp4, ContentTypeVideoMpeg, ContentTypeVideoWebm, ContentTypeVideoAvi, ContentTypeVideoMov,
		ContentTypeTextPlain, ContentTypeTextHtml, ContentTypeTextCss, ContentTypeTextJavascript, ContentTypeTextMarkdown, ContentTypeTextCsv,
		ContentTypeApplicationJson, ContentTypeApplicationXml, ContentTypeApplicationPdf, ContentTypeApplicationZip, ContentTypeApplicationGzip,
		ContentTypeApplicationExcelXlsx, ContentTypeApplicationWordDocx, ContentTypeApplicationPowerpointPptx, ContentTypeApplicationFormUrlencoded, ContentTypeApplicationOctetStream, ContentTypeApplicationRtf, ContentTypeApplicationJavascript,
	)
}

func (c ContentType) Validate() errx.Error {
	if c.Enum().Contains(c) {
		return nil
	}
	return errx.Validation.WithMsgf("invalid content type %s", c).Err()
}

type Method string

const (
	MethodGet    Method = "GET"
	MethodPost   Method = "POST"
	MethodPut    Method = "PUT"
	MethodPatch  Method = "PATCH"
	MethodDelete Method = "DELETE"
)

func (m Method) Enum() types.Enum {
	return types.RegisterEnum(MethodGet, MethodPost, MethodPut, MethodPatch, MethodDelete)
}

type HeaderParams struct {
	Authorization  string       `header:"Authorization,omitempty"`
	UserAgent      string       `header:"User-Agent,omitempty"`
	AcceptLanguage string       `header:"Accept-Language,omitempty"`
	Locale         types.Locale `header:"X-Locale,omitempty"`
	Timestamp      int64        `header:"X-Timestamp,omitempty"`
	RequestID      string       `header:"X-Request-Id,omitempty"`
	DeviceID       string       `header:"X-Device-Id,omitempty"`
	RealIP         string       `header:"X-Real-IP,omitempty"`
	IPCountry      string       `header:"X-IP-Country,omitempty"`
	IPRegion       string       `header:"X-IP-Region,omitempty"`
	IPCity         string       `header:"X-IP-City,omitempty"`
	Sign           string       `header:"X-Sign,omitempty"`
}

type ErrorDetails struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Type    errx.Type `json:"type"`
}
