package ibmmqlib

import (
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/felix-lessoer/qbeat/beater/ibmmq"
)

type Response struct {
	TargetObject string
	Metricset    string
	Metrictype   string
	MetricName   string
	Values       map[string]interface{}
}

var (
	err error
)

func connectLegacy(qMgrName string, remoteQMgrName string) error {
	qMgr, err := ibmmq.Conn(qMgrName)

	logp.Info("Connect to command queue")
	//Connect to Command Queue
	mqod := ibmmq.NewMQOD()
	openOptions := ibmmq.MQOO_OUTPUT | ibmmq.MQOO_FAIL_IF_QUIESCING
	mqod.ObjectType = ibmmq.MQOT_Q
	mqod.ObjectName = "SYSTEM.ADMIN.COMMAND.QUEUE"

	cmdQObj, err = qMgr.Open(mqod, openOptions)

	if err != nil {
		return err
	}

	logp.Info("Connect to Reply queue")
	//Connect to Reply Queue
	mqod2 := ibmmq.NewMQOD()
	openOptions2 := ibmmq.MQOO_INPUT_AS_Q_DEF | ibmmq.MQOO_FAIL_IF_QUIESCING
	mqod2.ObjectType = ibmmq.MQOT_Q
	mqod2.ObjectName = "SYSTEM.DEFAULT.MODEL.QUEUE"
	replyQObj, err = qMgr.Open(mqod2, openOptions2)

	if err != nil {
		return err
	}

	qmgrConnected = true

	return err
}

func getQueueStatistics(localQueueName string) (map[string]*Response, error) {
	var params = make(map[int32]string)
	params[ibmmq.MQCA_Q_NAME] = localQueueName

	err = putCommand(ibmmq.MQCMD_RESET_Q_STATS, params)
	return parseResponse()
}

func getQueueStatus(localQueueName string) (map[string]*Response, error) {
	var params = make(map[int32]string)
	params[ibmmq.MQCA_Q_NAME] = localQueueName

	err = putCommand(ibmmq.MQCMD_INQUIRE_Q_STATUS, params)
	return parseResponse()
}

func getQueueMetadata(localQueueName string) (map[string]*Response, error) {
	var params = make(map[int32]string)
	params[ibmmq.MQCA_Q_NAME] = localQueueName

	err = putCommand(ibmmq.MQCMD_INQUIRE_Q, params)
	return parseResponse()
}

func getChannelMetadata(channelName string) (map[string]*Response, error) {
	var params = make(map[int32]string)
	params[ibmmq.MQCACH_CHANNEL_NAME] = channelName

	err = putCommand(ibmmq.MQCMD_INQUIRE_CHANNEL, params)
	return parseResponse()
}

func getChannelStatus(channelName string) (map[string]*Response, error) {
	var params = make(map[int32]string)
	params[ibmmq.MQCACH_CHANNEL_NAME] = channelName

	err = putCommand(ibmmq.MQCMD_INQUIRE_CHANNEL_STATUS, params)
	return parseResponse()
}

func getQManagerMetadata() (map[string]*Response, error) {
	var params = make(map[int32]string)

	err = putCommand(ibmmq.MQCMD_INQUIRE_Q_MGR, params)
	return parseResponse()
}

func getQManagerStatus() (map[string]*Response, error) {
	var params = make(map[int32]string)

	err = putCommand(ibmmq.MQCMD_INQUIRE_Q_MGR_STATUS, params)
	return parseResponse()
}

func getAdvancedResponse(cmdString string, paramsInput map[string]interface{}) (map[string]*Response, error) {
	var params = make(map[int32]string)
	var cmd int32

	cmd = GetMQConstant(cmdString)
	for paramNameString, paramValue := range paramsInput {
		params[GetMQConstant(paramNameString)] = paramValue.(string)
	}

	err = putCommand(cmd, params)
	return parseResponse()
}

/***
Translates a value that is delivered as number to a string
*/
func translateValue(key string, value int64) string {
	var mapping = make(map[string]string)
	mapping["mqiach_channel_status"] = "CHS"
	mapping["mqiach_channel_type"] = "CHT"
	mapping["mqia_q_type"] = "QT"
	mapping["mqia_npm_class"] = "NPM"

	if mapping[key] != "" {
		returnName := ibmmq.MQItoString(mapping[key], int(value))
		returnName = strings.ToLower(returnName)
		return returnName
	}
	return ""
}

func parseResponse() (map[string]*Response, error) {
	var buf = make([]byte, 131072)
	var elem *ibmmq.PCFParameter
	var responses map[string]*Response
	var resp *Response
	responses = make(map[string]*Response)
	// Loop here to get every message in the queue

	var count = 0
	for err == nil {
		count = count + 1
		resp = new(Response)
		resp.Values = make(map[string]interface{})
		buf, err = GetMessageWithHObj(true, replyQObj)
		elemList, _ := ParsePCFResponse(buf)

		if err != nil {
			mqreturn := err.(*ibmmq.MQReturn)

			if mqreturn.MQCC == ibmmq.MQCC_FAILED && mqreturn.MQRC != ibmmq.MQRC_NO_MSG_AVAILABLE {
				logp.Debug("", "Error %v", err)
				return nil, err
			}
		}
		//Set sequential id for initalization of the id in case that there is no other identifier
		var key = strconv.Itoa(count)
		for i := 0; i < len(elemList); i++ {
			elem = elemList[i]
			logp.Debug("", "Current parameter (Type: %v): %v ", ibmmq.MQItoString("CFT", int(elem.Type)), normalizeMetricNames(elem.Parameter))
			switch elem.Parameter {
			case ibmmq.MQCA_Q_NAME:
				for i := 0; i < len(elem.String); i++ {
					logp.Debug("", "Current queue %v", strings.TrimSpace(elem.String[i]))
					resp.TargetObject = strings.TrimSpace(elem.String[i])
					resp.Metricset = "Queue"
					resp.Metrictype = "Status"
					key = resp.TargetObject
				}
			case ibmmq.MQCACH_CHANNEL_NAME:
				for i := 0; i < len(elem.String); i++ {
					logp.Debug("", "Current channel %v", strings.TrimSpace(elem.String[i]))
					resp.TargetObject = strings.TrimSpace(elem.String[i])
					resp.Metricset = "Channel"
					resp.Metrictype = "Status"
					key = resp.TargetObject
				}
			case ibmmq.MQCA_Q_MGR_NAME:
				for i := 0; i < len(elem.String); i++ {
					logp.Debug("", "Current queueManager %v", strings.TrimSpace(elem.String[i]))
					resp.TargetObject = strings.TrimSpace(elem.String[i])
					resp.Metricset = "QueueManager"
					resp.Metrictype = "Status"
					key = resp.TargetObject
				}
			default:
				if normalizeMetricNames(elem.Parameter) != "" {
					paramName := normalizeMetricNames(elem.Parameter)
					switch elem.Type {
					case ibmmq.MQCFT_INTEGER:
						resp.Values[paramName] = elem.Int64Value[0]
						logp.Debug("", "Try to translate %v: %v", paramName, elem.Int64Value[0])
						strValue := translateValue(paramName, elem.Int64Value[0])
						if strValue != "" {
							resp.Values[paramName+"_str"] = strValue
							logp.Debug("", "Translation successfull")
						}
					case ibmmq.MQCFT_INTEGER_LIST:
						for k, v := range elem.Int64Value {
							resp.Values[paramName+"_"+strconv.Itoa(k)] = v
						}
					case ibmmq.MQCFT_STRING:
						resp.Values[paramName] = strings.TrimSpace(elem.String[0])
					default:
						logp.Debug("", "Unhandeled parameter: %v type %v", normalizeMetricNames(elem.Parameter), ibmmq.MQItoString("CFT", int(elem.Type)))
					}
				}
			}
		}
		if key != "" {
			responses[key] = resp
		}
	}
	//Reset err if error is no more messages

	return responses, nil

}

func putCommand(commandCode int32, params map[int32]string) error {
	var buf []byte

	//Insert command
	putmqmd := ibmmq.NewMQMD()
	pmo := ibmmq.NewMQPMO()

	putmqmd.Format = "MQADMIN"
	putmqmd.ReplyToQ = replyQObj.Name
	putmqmd.MsgType = ibmmq.MQMT_REQUEST
	putmqmd.Report = ibmmq.MQRO_PASS_DISCARD_AND_EXPIRY

	// Reset QStats
	cfh := ibmmq.NewMQCFH()
	cfh.Command = commandCode

	logp.Info("%v initiated", ibmmq.MQItoString("CMD", int(commandCode)))
	// Add the parameters once at a time into a buffer
	for paramType, paramValue := range params {
		if paramType != 0 {
			logp.Info("Param %v set to %v", ibmmq.MQItoString("CA", int(paramType)), paramValue)
			pcfparm := new(ibmmq.PCFParameter)
			pcfparm.Type = ibmmq.MQCFT_STRING
			pcfparm.Parameter = paramType
			pcfparm.String = []string{paramValue}
			cfh.ParameterCount++
			buf = append(buf, pcfparm.Bytes()...)
		}
	}
	buf = append(cfh.Bytes(), buf...)

	// And put the command to the queue
	err = cmdQObj.Put(putmqmd, pmo, buf)

	if err != nil {
		logp.Info("Error putting the command into command queue")
		logp.Info("%v", err)
	}

	return err
}

func normalizeMetricNames(parameter int32) string {
	var returnName string
	returnName = ibmmq.MQItoString("IA", int(parameter))
	if returnName == "" {
		returnName = ibmmq.MQItoString("CA", int(parameter))
	}
	returnName = strings.ToLower(returnName)

	return returnName
}
