/*
 * Copyright 2025 superagent-ai Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const hrDefaultTimeout = 30 * time.Second

type hrClient struct {
	client  *http.Client
	baseURL string

	mu             sync.Mutex
	lastToken      string
	sessionCreated bool
}

func newHRClient() *hrClient {
	baseURL := os.Getenv("HR_API_BASE_URL")
	if baseURL == "" {
		baseURL = "https://hrssctest.haier.net/hrssc"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	jar, _ := cookiejar.New(nil)
	return &hrClient{
		client:  &http.Client{Timeout: hrDefaultTimeout, Jar: jar},
		baseURL: baseURL,
	}
}

// ensureSession calls loginCallback to exchange the access_token for a session cookie.
// HRSSC requires a valid HRSSC_sessionKey cookie for all API calls.
func (c *hrClient) ensureSession(ctx context.Context, token string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sessionCreated && c.lastToken == token {
		return nil
	}

	loginURL := c.baseURL + "/feishuNew/loginCallback?accessToken=" + url.QueryEscape(token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, loginURL, nil)
	if err != nil {
		return fmt.Errorf("loginCallback request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("loginCallback execute: %w", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("loginCallback returned HTTP %d", resp.StatusCode)
	}

	// Also set access_token as cookie (the browser frontend does this implicitly)
	parsedURL, _ := url.Parse(c.baseURL)
	c.client.Jar.SetCookies(parsedURL, []*http.Cookie{
		{Name: "access_token", Value: token},
	})

	c.lastToken = token
	c.sessionCreated = true
	return nil
}

func (c *hrClient) doJSON(ctx context.Context, path string, token string, body interface{}) (string, error) {
	if err := c.ensureSession(ctx, token); err != nil {
		return "", err
	}
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("access-token", token)
	}
	return c.execute(req)
}

func (c *hrClient) doForm(ctx context.Context, path string, token string, params url.Values) (string, error) {
	if err := c.ensureSession(ctx, token); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(params.Encode()))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if token != "" {
		req.Header.Set("access-token", token)
	}
	return c.execute(req)
}

func (c *hrClient) doMultipartForm(ctx context.Context, path string, token string, params map[string]string) (string, error) {
	if err := c.ensureSession(ctx, token); err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range params {
		if err := w.WriteField(k, v); err != nil {
			return "", fmt.Errorf("write field %s: %w", k, err)
		}
	}
	w.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("access-token", token)
	}
	return c.execute(req)
}

func (c *hrClient) execute(req *http.Request) (string, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		c.mu.Lock()
		c.sessionCreated = false
		c.mu.Unlock()
		return "", fmt.Errorf("session expired, got login page (HTTP %d)", resp.StatusCode)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return string(respBody), nil
}

func newHRSettlementTools() []tool.InvokableTool {
	hc := newHRClient()
	return []tool.InvokableTool{
		&hrGetUserInfoTool{hc: hc},
		&hrGetFlowStatusTool{hc: hc},
		&hrGetFlowProgressTool{hc: hc},
		&hrCheckEducationTool{hc: hc},
		&hrGetSettlementOptionsTool{hc: hc},
		&hrSubmitRegistrationTool{hc: hc},
		&hrGetRegistrationDetailsTool{hc: hc},
		&hrSubmitApplicationTool{hc: hc},
		&hrGetApplicationDetailsTool{hc: hc},
		&hrGetPoliceStationTool{hc: hc},
		&hrGetHukouInfoTool{hc: hc},
		&hrSubmitHukouChangeTool{hc: hc},
		&hrCheckHukouChangeStatusTool{hc: hc},
	}
}

// ==================== Tool 1: hr_get_user_info ====================

var _ tool.InvokableTool = (*hrGetUserInfoTool)(nil)

type hrGetUserInfoTool struct{ hc *hrClient }

func (t *hrGetUserInfoTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_get_user_info",
		Desc: "获取当前员工的基本信息（工号、姓名、部门、手机号等），落户流程第1步。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrGetUserInfoTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_get_user_info: parse args: %w", err)
	}
	return t.hc.doJSON(ctx, "/user/getAllUserInfo", args.AccessToken, nil)
}

// ==================== Tool 2: hr_get_flow_status ====================

var _ tool.InvokableTool = (*hrGetFlowStatusTool)(nil)

type hrGetFlowStatusTool struct{ hc *hrClient }

func (t *hrGetFlowStatusTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_get_flow_status",
		Desc: "查询用户落户流程的当前节点状态列表，包含每个步骤的状态、操作人和时间。返回所有9个步骤的状态。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
			"user_code": {
				Desc:     "用户工号（empNo），从 hr_get_user_info 返回的 empNo 获取",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrGetFlowStatusTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken string `json:"access_token"`
		UserCode    string `json:"user_code"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_get_flow_status: parse args: %w", err)
	}
	params := map[string]string{"userCode": args.UserCode}
	return t.hc.doMultipartForm(ctx, "/businesshall/businessinfo/getAIToDeal", args.AccessToken, params)
}

// ==================== Tool 3: hr_get_flow_progress ====================

var _ tool.InvokableTool = (*hrGetFlowProgressTool)(nil)

type hrGetFlowProgressTool struct{ hc *hrClient }

func (t *hrGetFlowProgressTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_get_flow_progress",
		Desc: "查询落户流程的整体进度信息，返回当前步骤编号和状态。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
			"emp_no": {
				Desc:     "员工工号",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrGetFlowProgressTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken string `json:"access_token"`
		EmpNo       string `json:"emp_no"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_get_flow_progress: parse args: %w", err)
	}
	body := map[string]string{"empNo": args.EmpNo}
	return t.hc.doJSON(ctx, "/eRecord/getLhByInit", args.AccessToken, body)
}

// ==================== Tool 4: hr_check_education ====================

var _ tool.InvokableTool = (*hrCheckEducationTool)(nil)

type hrCheckEducationTool struct{ hc *hrClient }

func (t *hrCheckEducationTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_check_education",
		Desc: "检查员工学历是否满足落户资格要求。返回 isJx=true 表示符合条件，isJx=false 表示不符合。这是报名（步骤2）的前置检查。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrCheckEducationTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_check_education: parse args: %w", err)
	}
	return t.hc.doJSON(ctx, "/eRecord/getJxByEmpCode", args.AccessToken, nil)
}

// ==================== Tool 5: hr_get_settlement_options ====================

var _ tool.InvokableTool = (*hrGetSettlementOptionsTool)(nil)

type hrGetSettlementOptionsTool struct{ hc *hrClient }

func (t *hrGetSettlementOptionsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_get_settlement_options",
		Desc: "获取落户条件的下拉选项列表（如学历类型、落户条件等）。用于填写报名表单时获取可选值。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
			"id": {
				Desc:     "字典分类ID，获取落户条件选项时传 'lhCondition'",
				Type:     schema.String,
				Required: true,
			},
			"yq_id": {
				Desc:     "园区ID（可选），用于过滤特定园区的选项",
				Type:     schema.String,
				Required: false,
			},
		}),
	}, nil
}

func (t *hrGetSettlementOptionsTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken string `json:"access_token"`
		ID          string `json:"id"`
		YqID        string `json:"yq_id"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_get_settlement_options: parse args: %w", err)
	}
	params := url.Values{}
	params.Set("id", args.ID)
	if args.YqID != "" {
		params.Set("yqId", args.YqID)
	}
	return t.hc.doForm(ctx, "/eRecord/getListDictionaryByIdAndYqId", args.AccessToken, params)
}

// ==================== Tool 6: hr_submit_registration ====================

var _ tool.InvokableTool = (*hrSubmitRegistrationTool)(nil)

type hrSubmitRegistrationTool struct{ hc *hrClient }

func (t *hrSubmitRegistrationTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_submit_registration",
		Desc: "提交落户报名表单（步骤2）。成功后返回 orderCode，后续所有步骤都需要这个编号。提交前请确保已通过 hr_check_education 验证学历资格。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
			"settle_condition": {
				Desc:     "落户条件编码，从 hr_get_settlement_options 获取",
				Type:     schema.String,
				Required: true,
			},
			"emp_no": {
				Desc:     "员工工号",
				Type:     schema.String,
				Required: true,
			},
			"phone": {
				Desc:     "手机号",
				Type:     schema.String,
				Required: true,
			},
			"id_card": {
				Desc:     "身份证号",
				Type:     schema.String,
				Required: true,
			},
			"current_address": {
				Desc:     "当前户籍所在地",
				Type:     schema.String,
				Required: true,
			},
			"current_police": {
				Desc:     "户口所在地派出所",
				Type:     schema.String,
				Required: true,
			},
			"has_bought_house": {
				Desc:     "青岛是否购房：0=否，1=是。如果为1则不符合集体户条件。",
				Type:     schema.String,
				Required: true,
			},
			"has_property_cert": {
				Desc:     "是否有房产证：0=否，1=是。如果为1则不符合集体户条件。",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrSubmitRegistrationTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken     string `json:"access_token"`
		SettleCondition string `json:"settle_condition"`
		EmpNo           string `json:"emp_no"`
		Phone           string `json:"phone"`
		IDCard          string `json:"id_card"`
		CurrentAddress  string `json:"current_address"`
		CurrentPolice   string `json:"current_police"`
		HasBoughtHouse  string `json:"has_bought_house"`
		HasPropertyCert string `json:"has_property_cert"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_submit_registration: parse args: %w", err)
	}

	if args.HasBoughtHouse == "1" || args.HasPropertyCert == "1" {
		return `{"success":false,"message":"青岛已购房或有房产证的员工不符合集体户落户条件"}`, nil
	}

	params := url.Values{}
	params.Set("settleCondition", args.SettleCondition)
	params.Set("empNo", args.EmpNo)
	params.Set("phone", args.Phone)
	params.Set("idCard", args.IDCard)
	params.Set("currentAddress", args.CurrentAddress)
	params.Set("currentPolice", args.CurrentPolice)
	params.Set("hasBoughtHouse", args.HasBoughtHouse)
	params.Set("hasPropertyCert", args.HasPropertyCert)
	return t.hc.doForm(ctx, "/eRecord/baomingByLh", args.AccessToken, params)
}

// ==================== Tool 7: hr_get_registration_details ====================

var _ tool.InvokableTool = (*hrGetRegistrationDetailsTool)(nil)

type hrGetRegistrationDetailsTool struct{ hc *hrClient }

func (t *hrGetRegistrationDetailsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_get_registration_details",
		Desc: "查看已提交的报名详情，包括报名表单填写的所有信息。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
			"order_code": {
				Desc:     "流程编号（报名成功后返回的 orderCode）",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrGetRegistrationDetailsTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken string `json:"access_token"`
		OrderCode   string `json:"order_code"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_get_registration_details: parse args: %w", err)
	}
	params := map[string]string{"orderCode": args.OrderCode}
	return t.hc.doMultipartForm(ctx, "/eRecord/getLhBaoMingByOrderCode", args.AccessToken, params)
}

// ==================== Tool 8: hr_submit_application ====================

var _ tool.InvokableTool = (*hrSubmitApplicationTool)(nil)

type hrSubmitApplicationTool struct{ hc *hrClient }

func (t *hrSubmitApplicationTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_submit_application",
		Desc: "提交落户申请（步骤5），包含已上传文件的信息。文件需要先通过前端上传获取fileUuid，再将文件列表以JSON格式传入。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
			"order_code": {
				Desc:     "流程编号",
				Type:     schema.String,
				Required: true,
			},
			"json_list_str": {
				Desc:     "文件列表JSON字符串，格式：[{\"type\":\"L6\",\"fileUuid\":\"xxx\",\"fileName\":\"身份证正面.jpg\"}]。类型编码：L1=户口首页,L2=个人页,L3=索引页,L4=迁移证,L5=户籍证明,L6=身份证正面,L7=身份证反面,L8=人才审核截图",
				Type:     schema.String,
				Required: true,
			},
			"registered_outside_province": {
				Desc:     "是否省外户籍：0=省内，1=省外",
				Type:     schema.String,
				Required: true,
			},
			"online_migration": {
				Desc:     "是否线上迁移：0=线下，1=线上",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrSubmitApplicationTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken               string `json:"access_token"`
		OrderCode                 string `json:"order_code"`
		JSONListStr               string `json:"json_list_str"`
		RegisteredOutsideProvince string `json:"registered_outside_province"`
		OnlineMigration           string `json:"online_migration"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_submit_application: parse args: %w", err)
	}
	params := map[string]string{
		"orderCode":                 args.OrderCode,
		"jsonListStr":               args.JSONListStr,
		"registeredOutsideProvince": args.RegisteredOutsideProvince,
		"onlineMigration":           args.OnlineMigration,
	}
	return t.hc.doMultipartForm(ctx, "/eRecord/lhsq", args.AccessToken, params)
}

// ==================== Tool 9: hr_get_application_details ====================

var _ tool.InvokableTool = (*hrGetApplicationDetailsTool)(nil)

type hrGetApplicationDetailsTool struct{ hc *hrClient }

func (t *hrGetApplicationDetailsTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_get_application_details",
		Desc: "查看落户申请详情，包括已上传的文件列表和审核状态。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
			"order_code": {
				Desc:     "流程编号",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrGetApplicationDetailsTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken string `json:"access_token"`
		OrderCode   string `json:"order_code"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_get_application_details: parse args: %w", err)
	}
	params := map[string]string{"orderCode": args.OrderCode}
	return t.hc.doMultipartForm(ctx, "/eRecord/getLuohuDetailsByOrderCode", args.AccessToken, params)
}

// ==================== Tool 10: hr_get_police_station ====================

var _ tool.InvokableTool = (*hrGetPoliceStationTool)(nil)

type hrGetPoliceStationTool struct{ hc *hrClient }

func (t *hrGetPoliceStationTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_get_police_station",
		Desc: "获取落户对应的派出所信息（步骤7需要），包括派出所名称、地址等。用户需要到该派出所办理户口迁移。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
			"order_code": {
				Desc:     "流程编号",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrGetPoliceStationTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken string `json:"access_token"`
		OrderCode   string `json:"order_code"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_get_police_station: parse args: %w", err)
	}
	params := map[string]string{"orderCode": args.OrderCode}
	return t.hc.doMultipartForm(ctx, "/eRecord/getPcsByOrderCode", args.AccessToken, params)
}

// ==================== Tool 11: hr_get_hukou_info ====================

var _ tool.InvokableTool = (*hrGetHukouInfoTool)(nil)

type hrGetHukouInfoTool struct{ hc *hrClient }

func (t *hrGetHukouInfoTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_get_hukou_info",
		Desc: "获取员工当前的户口信息，包括户口所在地、户口性质等。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrGetHukouInfoTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_get_hukou_info: parse args: %w", err)
	}
	params := map[string]string{}
	return t.hc.doMultipartForm(ctx, "/eRecord/getUserHuKouInfo", args.AccessToken, params)
}

// ==================== Tool 12: hr_submit_hukou_change ====================

var _ tool.InvokableTool = (*hrSubmitHukouChangeTool)(nil)

type hrSubmitHukouChangeTool struct{ hc *hrClient }

func (t *hrSubmitHukouChangeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_submit_hukou_change",
		Desc: "提交户口性质变更申请（步骤8），在完成户口迁移后上传新户口单页并申请变更。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
			"order_code": {
				Desc:     "流程编号",
				Type:     schema.String,
				Required: true,
			},
			"hukou_address": {
				Desc:     "户口落户地址",
				Type:     schema.String,
				Required: true,
			},
			"move_in_date": {
				Desc:     "迁入日期，格式：yyyy-MM-dd",
				Type:     schema.String,
				Required: true,
			},
			"file_uuid": {
				Desc:     "户口单页文件的UUID（先通过前端上传获取）",
				Type:     schema.String,
				Required: true,
			},
			"file_name": {
				Desc:     "户口单页文件名",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrSubmitHukouChangeTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken  string `json:"access_token"`
		OrderCode    string `json:"order_code"`
		HukouAddress string `json:"hukou_address"`
		MoveInDate   string `json:"move_in_date"`
		FileUUID     string `json:"file_uuid"`
		FileName     string `json:"file_name"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_submit_hukou_change: parse args: %w", err)
	}
	params := map[string]string{
		"orderCode":      args.OrderCode,
		"categoryAddress": args.HukouAddress,
		"luohuMoveInDate": args.MoveInDate,
		"fileUuid":        args.FileUUID,
		"fileName":        args.FileName,
		"type":            "L9",
	}
	return t.hc.doMultipartForm(ctx, "/eRecord/hukouTypeApply", args.AccessToken, params)
}

// ==================== Tool 13: hr_check_hukou_change_status ====================

var _ tool.InvokableTool = (*hrCheckHukouChangeStatusTool)(nil)

type hrCheckHukouChangeStatusTool struct{ hc *hrClient }

func (t *hrCheckHukouChangeStatusTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hr_check_hukou_change_status",
		Desc: "查看户口性质变更申请的处理状态。",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"access_token": {
				Desc:     "用户的认证token",
				Type:     schema.String,
				Required: true,
			},
			"order_code": {
				Desc:     "流程编号",
				Type:     schema.String,
				Required: true,
			},
		}),
	}, nil
}

func (t *hrCheckHukouChangeStatusTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		AccessToken string `json:"access_token"`
		OrderCode   string `json:"order_code"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hr_check_hukou_change_status: parse args: %w", err)
	}
	params := map[string]string{"orderCode": args.OrderCode}
	return t.hc.doMultipartForm(ctx, "/eRecord/checkHuKouTypeApply", args.AccessToken, params)
}
