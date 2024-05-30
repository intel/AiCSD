/*********************************************************************
 * Copyright (c) Intel Corporation 2024
 * SPDX-License-Identifier: BSD-3-Clause
 **********************************************************************/

package functions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hybridgroup/mjpeg"
	"gocv.io/x/gocv"
	"google.golang.org/grpc"
	"image"
	"image/color"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"aicsd/as-pipeline-grpc-go/config"
	"aicsd/as-pipeline-grpc-go/pkg/ovms"
	"aicsd/as-pipeline-grpc-go/pkg/yolov5"
	grpc_client "aicsd/grpc-client"
	"aicsd/pkg"
	"aicsd/pkg/helpers"
	"aicsd/pkg/types"
	"aicsd/pkg/werrors"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
)

const (
	RETRY        = time.Millisecond
	MAX_RETRY    = 10
	ResourceName = "PipelineParameters"
	ModelName    = "yolov5s"
	ModelVersion = ""
)

// PipelineGrpcGo provides the data for the App Function Pipelines in this package
type PipelineGrpcGo struct {
	params          *PipelineParams
	lc              logger.LoggingClient
	outputFiles     []types.OutputFile
	pipelineResults string
	qcFlags         string
	config          *config.Configuration
	GrpcClient      grpc_client.GRPCInferenceServiceClient
	GrpcConn        *grpc.ClientConn
	CvParams        *GoCVParams
	outputStream    string
}

type GoCVParams struct {
	camHeight float32
	camWidth  float32
	img       *gocv.Mat
	webcam    *gocv.VideoCapture
	Stream    *mjpeg.Stream
}

func NewPipelineGrpcGo(pipelineResults string, qcFlags string, config *config.Configuration) PipelineGrpcGo {
	// // Connect to gRPC server
	conn, err := grpc.Dial(config.OvmsUrl, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Couldn't connect to endpoint %s: %v", config.OvmsUrl, err)
	}

	client := grpc_client.NewGRPCInferenceServiceClient(conn)

	// create the mjpeg stream
	cvParams := GoCVParams{}
	cvParams.Stream = mjpeg.NewStream()

	return PipelineGrpcGo{
		pipelineResults: pipelineResults,
		qcFlags:         qcFlags,
		config:          config,
		GrpcConn:        conn,
		GrpcClient:      client,
		CvParams:        &cvParams,
		outputStream:    config.OutputStreamHost,
	}
}

func (p *PipelineGrpcGo) CloseGrpcConnection() {
	p.GrpcConn.Close()
}

// PipelineParams is the data expected in the EdgeX Reading objectValue
type PipelineParams struct {
	InputFileLocation string
	OutputFileFolder  string
	ModelParams       map[string]string
	JobUpdateUrl      string // already includes the parameterized jobid and taskid values
	PipelineStatusUrl string // already includes the parameterized jobid and taskid values
}

// ProcessEvent is the entry point App Pipeline Function that receives and processes the EdgeX Event/Reading that
// contains the Pipeline Parameters
func (p *PipelineGrpcGo) ProcessEvent(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc = ctx.LoggingClient()
	p.lc.Debugf("Running ProcessEvent...")

	if data == nil {
		err := fmt.Errorf("no data received")
		p.lc.Errorf("ProcessEvent failed: %s", err.Error())
		return false, err // Terminate since can not send back status w/o knowing the URL that is passed in the data
	}

	event, ok := data.(dtos.Event)
	if !ok {
		err := fmt.Errorf("type received is not an Event")
		p.lc.Errorf("ProcessEvent failed: %s", err.Error())
		return false, err // Terminate since can not send back status w/o knowing the URL that is passed in the data
	}

	err := helpers.AppFunctionEventValidation(event, ResourceName, ResourceName)
	if err != nil {
		return true, werrors.WrapMsg(err, "failed AppFunctionEventValidation")
	}

	jsonData, err := json.Marshal(event.Readings[0].ObjectValue)
	if err != nil {
		err = fmt.Errorf("unable to marshal Object Value back to JSON: %s", err.Error())
		p.lc.Errorf("ProcessEvent failed: %s", err.Error())
		return false, err // Terminate since can not send back status w/o knowing the URL that is passed in the data
	}

	params := PipelineParams{}
	if err = json.Unmarshal(jsonData, &params); err != nil {
		err = fmt.Errorf("unable to unmarshal Object Value back from JSON to struct: %s", err.Error())
		p.lc.Errorf("ProcessEvent failed: %s", err.Error())
		return false, err // Terminate since can not send back status w/o knowing the URL that is passed in the data
	}
	if strings.Contains(params.InputFileLocation, "rtsp") && !strings.Contains(params.InputFileLocation, "//") {
		params.InputFileLocation = strings.Replace(params.InputFileLocation, "rtsp:/", "rtsp://", 1)
	}
	p.params = &params

	p.lc.Debugf("Pipeline %s: Received the following PipelineParams: %+v", ctx.PipelineId(), params)

	return true, data // All is good, this indicates success for the next function
}

func (p *PipelineGrpcGo) SetupStreamConfig(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	var err error
	// Previous function returns any error as its result which is passed as `data` to this function.
	if pipelineErr, ok := data.(error); ok {
		p.lc.Info("SetupStreamConfig: Forwarding error to next function")
		return true, pipelineErr // Keep passing the error forward inorder for to make it to the ReportStatus pipeline function
	}
	// start the webcam
	p.lc.Debugf("SetupStreamConfig: opening video capture from %s", p.params.InputFileLocation)
	p.CvParams.webcam, err = gocv.OpenVideoCapture(p.params.InputFileLocation) //  /dev/video4
	if err != nil {
		errMsg := fmt.Errorf("failed to open device: %s", p.params.InputFileLocation)
		fmt.Println(errMsg)
	}

	p.CvParams.camHeight = float32(p.CvParams.webcam.Get(gocv.VideoCaptureFrameHeight))
	p.CvParams.camWidth = float32(p.CvParams.webcam.Get(gocv.VideoCaptureFrameWidth))

	img := gocv.NewMat()
	p.CvParams.img = &img

	return true, data
}

// RunOvmsModel is an App Pipeline Function that calls the OVMS model
func (p *PipelineGrpcGo) RunOvmsModel(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc.Debugf("Running CallOvmsModel...")

	defer p.CvParams.webcam.Close()
	defer p.CvParams.img.Close()

	var aggregateLatencyAfterInfer, frameNum float64

	// Previous function returns any error as its result which is passed as `data` to this function.
	if pipelineErr, ok := data.(error); ok {
		p.lc.Info("CallOvmsModel: Forwarding error to next function")
		return true, pipelineErr // Keep passing the error forward inorder for to make it to the ReportStatus pipeline function
	}
	retryCnt := 0
	initTime := float64(time.Now().UnixMilli())
	for p.CvParams.webcam.IsOpened() {
		if ok := p.CvParams.webcam.Read(p.CvParams.img); !ok {
			// retry once after 1 millisecond
			time.Sleep(RETRY)
			retryCnt++
			if retryCnt == MAX_RETRY {
				p.lc.Info("RunOvmsModel: webcam Read not ok for max retries")
				return true, data
			}
			continue
		}
		if p.CvParams.img.Empty() {
			retryCnt++
			if retryCnt == MAX_RETRY {
				p.lc.Info("RunOvmsModel: empty image for max retries")
				return true, data
			}
			continue
		}

		frameNum++

		start := float64(time.Now().UnixMilli())
		fp32Image := gocv.NewMat()
		defer fp32Image.Close()

		// resize image to yolov5 model specifications
		gocv.Resize(*p.CvParams.img, &fp32Image, image.Point{int(yolov5.Input_shape[1]), int(yolov5.Input_shape[2])}, 0, 0, 3)

		// convert to image matrix to use float32
		fp32Image.ConvertTo(&fp32Image, gocv.MatTypeCV32F)
		imgToBytes, _ := fp32Image.DataPtrFloat32()

		// retry if error found to ovms server
		var err error
		var inferResponse *ovms.TensorOutputs
		retryCnt = 0
		for {
			if retryCnt > MAX_RETRY {
				// there is something broken sending request to the model server, cannot continue...
				p.lc.Info("RunOvmsModel: infer request error after max retry count: ", MAX_RETRY, "exiting...")
				os.Exit(1)
			}
			// TODO: add a check to see if the server is live and that the model/version exists
			inferResponse, err = ovms.ModelInferRequest(p.GrpcClient, imgToBytes, ModelName, ModelVersion)
			if err != nil {
				retryCnt++
				p.lc.Infof("RunOvmsModel: infer request error %v retry count: %d ", err, retryCnt)
			} else {
				break
			}
		}

		afterInfer := float64(time.Now().UnixMilli())
		aggregateLatencyAfterInfer += afterInfer - start

		detectedObjects := yolov5.DetectedObjects{}

		// temp code:
		output := ovms.TensorOutputs{
			RawData:    [][]byte{(*inferResponse).RawData[0]},
			DataShapes: [][]int64{(*inferResponse).DataShapes[0]},
		}
		output.ParseRawData()
		err = detectedObjects.Postprocess(output, p.CvParams.camHeight, p.CvParams.camHeight)
		if err != nil {
			p.lc.Errorf("RunOvmsModel: post process failed: %v", err)
		}

		detectedObjects = detectedObjects.FinalPostProcessAdvanced()

		// Print after processing latency
		afterFinalProcess := float64(time.Now().UnixMilli())
		processTime := afterFinalProcess - start
		avgFps := frameNum / ((afterFinalProcess - initTime) / 1000.0)
		averageFPSStr := fmt.Sprintf("%v\n", avgFps)
		p.lc.Infof("RunOvmsModel: Processing time: %v ms; fps: %s", processTime, averageFPSStr)

		// add bounding boxes to resized image
		detectedObjects.AddBoxesToFrame(&fp32Image, color.RGBA{0, 255, 0, 0}, p.CvParams.camWidth, p.CvParams.camWidth)

		buf, _ := gocv.IMEncode(".jpg", fp32Image)
		p.CvParams.Stream.UpdateJPEG(buf.GetBytes())
		buf.Close()
		// TODO: send the detected objects to the message bus or update job structure to handle stream data
		p.pipelineResults = p.outputStream
	}

	return true, data // All is good, this indicates success for the next function
}

// UpdateJobRepo is an App Pipeline Function that does the PUT request to update the Job Repo with the Pipeline status details
func (p *PipelineGrpcGo) UpdateJobRepo(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc.Debugf("Running UpdateJobRepo...")

	pipelineStatus := struct {
		Status      string
		QCFlags     string
		OutputFiles []types.OutputFile
		Results     string
	}{
		Status:      pkg.TaskStatusComplete,
		QCFlags:     p.qcFlags,
		OutputFiles: p.outputFiles,
		Results:     p.pipelineResults,
	}

	// Previous function returns any error as its result which is passed as `data` to this function.
	if _, ok := data.(error); ok {
		pipelineStatus.Status = pkg.TaskStatusFailed
		pipelineStatus.Results = ""
		pipelineStatus.QCFlags = ""
		pipelineStatus.OutputFiles = []types.OutputFile{}
	}

	body, err := json.Marshal(pipelineStatus)
	if err != nil {
		err = werrors.WrapMsg(err, "failed to marshal Pipeline Status for updating Job Repo.")
		p.lc.Errorf("UpdateJobRepo failed: %s", err.Error())
		return true, err
	}

	request, err := http.NewRequest(http.MethodPut, p.params.JobUpdateUrl, bytes.NewBuffer(body))
	if err != nil {
		err = werrors.WrapMsg(err, "unable to create http request to update Job Repo.")
		p.lc.Errorf("UpdateJobRepo failed: %s", err.Error())
		return true, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		err = werrors.WrapMsg(err, "failed to send update Job Repo request.")
		p.lc.Errorf("UpdateJobRepo failed: %s", err.Error())
		return true, err
	}

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("update Job Repo request failed with status code %d", response.StatusCode)
		p.lc.Errorf("UpdateJobRepo failed: %s", err.Error())
		return true, err
	}

	p.lc.Debugf("Pipeline %s: Updated Pipeline details with '%+v' using URL %s", ctx.PipelineId(), pipelineStatus, p.params.JobUpdateUrl)

	return true, data // Pass data which was passed to this function (which could be error) on to next function
}

// ReportStatus is an App Pipeline Function that does the POST request to Task Launcher for PipelineStatus
func (p *PipelineGrpcGo) ReportStatus(ctx interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	p.lc.Debugf("Running ReportStatus...")

	// Previous function returns any error as its result which is passed as `data` to this function.
	status := pkg.TaskStatusComplete
	if _, ok := data.(error); ok {
		status = pkg.TaskStatusFailed
		p.lc.Errorf("Pipeline failed. Sending %s status to %s", status, p.params.PipelineStatusUrl)
	}

	request, err := http.NewRequest(http.MethodPost, p.params.PipelineStatusUrl, bytes.NewBuffer([]byte(status)))
	if err != nil {
		err = werrors.WrapMsg(err, "failed to create http request to send pipeline status.")
		p.lc.Errorf("ReportStatus failed: %s", err.Error())
		return false, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		err = werrors.WrapMsg(err, "failed to send pipeline status request.")
		p.lc.Errorf("ReportStatus failed: %s", err.Error())
		return false, err
	}

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("pipeline status request failed with status code %d", response.StatusCode)
		p.lc.Errorf("ReportStatus failed: %s", err.Error())
		return false, err
	}

	// Note: This is needed to clear the output files that the pipeline sim should process on future runs
	p.outputFiles = []types.OutputFile{}

	p.lc.Debugf("Pipeline %s: Pipeline Status complete using URL %s", ctx.PipelineId(), p.params.PipelineStatusUrl)

	return false, nil // All done
}
