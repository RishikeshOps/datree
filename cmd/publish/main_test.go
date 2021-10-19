package publish

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/datreeio/datree/bl/files"
	"github.com/datreeio/datree/bl/messager"
	"github.com/datreeio/datree/pkg/httpClient"
	"github.com/datreeio/datree/pkg/localConfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type LocalConfigMock struct {
	mock.Mock
}

func (lc *LocalConfigMock) GetLocalConfiguration() (*localConfig.ConfigContent, error) {
	lc.Called()
	return &localConfig.ConfigContent{CliId: "cli_id"}, nil
}

type MessagerMock struct {
	mock.Mock
}

func (m *MessagerMock) LoadVersionMessages(messages chan *messager.VersionMessage, cliVersion string) {
	go func() {
		messages <- &messager.VersionMessage{
			CliVersion:   "1.2.3",
			MessageText:  "version message mock",
			MessageColor: "green"}
		close(messages)
	}()

	m.Called(messages, cliVersion)
}

type PrinterMock struct {
	mock.Mock
}

func (p *PrinterMock) PrintMessage(messageText string, messageColor string) {
	p.Called(messageText, messageColor)
}

type PublishClientMock struct {
	mock.Mock
}

func (p *PublishClientMock) PublishPolicies(policiesConfiguration files.UnknownStruct, cliId string) (httpClient.Response, error) {
	args := p.Called(policiesConfiguration, cliId)
	return args.Get(0).(httpClient.Response), args.Error(1)
}

func TestPublishCommand(t *testing.T) {
	localConfigMock := &LocalConfigMock{}
	localConfigMock.On("GetLocalConfiguration")

	messagerMock := &MessagerMock{}
	messagerMock.On("LoadVersionMessages", mock.Anything, mock.Anything)

	printerMock := &PrinterMock{}
	printerMock.On("PrintMessage", mock.Anything, mock.Anything)

	publishClientMock := &PublishClientMock{}

	ctx := &PublishCommandContext{
		CliVersion:       "0.0.1",
		LocalConfig:      localConfigMock,
		Messager:         messagerMock,
		Printer:          printerMock,
		PublishCliClient: publishClientMock,
	}

	testPublishCommandSuccess(t, ctx, publishClientMock)
	testPublishCommandFailedYaml(t, ctx)
	testPublishCommandFailedSchema(t, ctx, publishClientMock)
}

func testPublishCommandSuccess(t *testing.T, ctx *PublishCommandContext, publishClientMock *PublishClientMock) {
	publishClientMock.On("PublishPolicies", mock.Anything, mock.Anything).Return(httpClient.Response{}, nil).Once()
	_, err := publish(ctx, "../../internal/fixtures/policyAsCode/valid-schema.yaml")
	assert.Equal(t, nil, err)
}

func testPublishCommandFailedYaml(t *testing.T, ctx *PublishCommandContext) {
	_, err := publish(ctx, "../../internal/fixtures/policyAsCode/invalid-yaml.yaml")
	assert.NotEqual(t, nil, err)
	assert.Equal(t, "yaml: line 2: did not find expected key", err.Error())
}

func testPublishCommandFailedSchema(t *testing.T, ctx *PublishCommandContext, publishClientMock *PublishClientMock) {
	publishFailedPayloadMock := []string{"first error", "second error"}
	errMessage := strings.Join(publishFailedPayloadMock, ",")
	publishFailedResponseMock := PublishFailedResponse{
		Code:    "mocked code",
		Message: errMessage,
		Payload: publishFailedPayloadMock,
	}

	mockedResponseBody, _ := json.Marshal(publishFailedResponseMock)
	mockedResponse := httpClient.Response{
		StatusCode: 400,
		Body:       mockedResponseBody,
	}

	publishClientMock.On("PublishPolicies", mock.Anything, mock.Anything).Return(mockedResponse, errors.New(errMessage)).Once()
	publishFailedRes, err := publish(ctx, "../../internal/fixtures/policyAsCode/invalid-schemas/duplicate-rule-id.yaml")
	assert.NotEqual(t, nil, err)
	assert.Equal(t, errMessage, err.Error())
	assert.Equal(t, publishFailedResponseMock, publishFailedRes)
}
