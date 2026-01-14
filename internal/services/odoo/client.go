package odoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/kolo/xmlrpc"
)

// Client represents an Odoo XML-RPC client
type Client struct {
	URL        string
	Database   string
	Username   string
	Password   string
	Uid        int
	CommonURL  string
	ObjectURL  string
	HttpClient *http.Client
}

// NewClient creates a new Odoo client
func NewClient(url, db, username, password string) *Client {
	return &Client{
		URL:        url,
		Database:   db,
		Username:   username,
		Password:   password,
		CommonURL:  fmt.Sprintf("%s/xmlrpc/2/common", url),
		ObjectURL:  fmt.Sprintf("%s/xmlrpc/2/object", url),
		HttpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Authenticate authenticates with Odoo and returns the user ID
func (c *Client) Authenticate() (int, error) {
	client, err := xmlrpc.NewClient(c.CommonURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create XML-RPC client: %w", err)
	}
	defer client.Close()

	args := []interface{}{c.Database, c.Username, c.Password, make([]interface{}, 0)}
	var uid int
	if err := client.Call("authenticate", args, &uid); err != nil {
		return 0, fmt.Errorf("authentication failed: %w", err)
	}

	c.Uid = uid
	return uid, nil
}

// SearchRead performs a generic search_read operation
// model: Odoo model name (e.g., "product.product")
// domain: search criteria
// fields: fields to fetch
// limit: max records
// offset: offset for pagination
// result: pointer to slice of structs with xmlrpc tags
func (c *Client) SearchRead(model string, domain []interface{}, fields []string, limit, offset int, result interface{}) error {
	client, err := xmlrpc.NewClient(c.ObjectURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create XML-RPC client: %w", err)
	}
	defer client.Close()

	args := []interface{}{
		c.Database,
		c.Uid,
		c.Password,
		model,
		"search_read",
		[]interface{}{domain},
		map[string]interface{}{
			"fields": fields,
			"limit":  limit,
			"offset": offset,
		},
	}

	// First, get raw result
	var rawResult []map[string]interface{}
	if err := client.Call("execute_kw", args, &rawResult); err != nil {
		return fmt.Errorf("failed to execute search_read: %w", err)
	}

	// Convert raw maps to target struct using reflection and JSON marshaling
	jsonData, err := json.Marshal(rawResult)
	if err != nil {
		return fmt.Errorf("failed to marshal raw result: %w", err)
	}

	if err := json.Unmarshal(jsonData, result); err != nil {
		return fmt.Errorf("failed to unmarshal into target: %w", err)
	}

	return nil
}

// Search performs a generic search operation and returns IDs
func (c *Client) Search(model string, domain []interface{}, limit, offset int) ([]int64, error) {
	client, err := xmlrpc.NewClient(c.ObjectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create XML-RPC client: %w", err)
	}
	defer client.Close()

	args := []interface{}{
		c.Database,
		c.Uid,
		c.Password,
		model,
		"search",
		[]interface{}{domain},
		map[string]interface{}{
			"limit":  limit,
			"offset": offset,
		},
	}

	var ids []int64
	if err := client.Call("execute_kw", args, &ids); err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}

	return ids, nil
}

// Read reads records by IDs
func (c *Client) Read(model string, ids []int64, fields []string, result interface{}) error {
	client, err := xmlrpc.NewClient(c.ObjectURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create XML-RPC client: %w", err)
	}
	defer client.Close()

	args := []interface{}{
		c.Database,
		c.Uid,
		c.Password,
		model,
		"read",
		[]interface{}{ids},
		map[string]interface{}{
			"fields": fields,
		},
	}

	var rawResult []map[string]interface{}
	if err := client.Call("execute_kw", args, &rawResult); err != nil {
		return fmt.Errorf("failed to execute read: %w", err)
	}

	// Convert to target struct
	jsonData, err := json.Marshal(rawResult)
	if err != nil {
		return fmt.Errorf("failed to marshal raw result: %w", err)
	}

	if err := json.Unmarshal(jsonData, result); err != nil {
		return fmt.Errorf("failed to unmarshal into target: %w", err)
	}

	return nil
}

// Create creates a new record
func (c *Client) Create(model string, values map[string]interface{}) (int64, error) {
	client, err := xmlrpc.NewClient(c.ObjectURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create XML-RPC client: %w", err)
	}
	defer client.Close()

	args := []interface{}{
		c.Database,
		c.Uid,
		c.Password,
		model,
		"create",
		[]interface{}{values},
	}

	var id int64
	if err := client.Call("execute_kw", args, &id); err != nil {
		return 0, fmt.Errorf("failed to create record: %w", err)
	}

	return id, nil
}

// Write updates existing record(s)
func (c *Client) Write(model string, ids []int64, values map[string]interface{}) error {
	client, err := xmlrpc.NewClient(c.ObjectURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create XML-RPC client: %w", err)
	}
	defer client.Close()

	args := []interface{}{
		c.Database,
		c.Uid,
		c.Password,
		model,
		"write",
		[]interface{}{ids, values},
	}

	var success bool
	if err := client.Call("execute_kw", args, &success); err != nil {
		return fmt.Errorf("failed to write record: %w", err)
	}

	if !success {
		return fmt.Errorf("write operation returned false")
	}

	return nil
}

// Delete (unlink) deletes record(s)
func (c *Client) Delete(model string, ids []int64) error {
	client, err := xmlrpc.NewClient(c.ObjectURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create XML-RPC client: %w", err)
	}
	defer client.Close()

	args := []interface{}{
		c.Database,
		c.Uid,
		c.Password,
		model,
		"unlink",
		[]interface{}{ids},
	}

	var success bool
	if err := client.Call("execute_kw", args, &success); err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	if !success {
		return fmt.Errorf("delete operation returned false")
	}

	return nil
}

// CallMethod calls a custom method on an Odoo model
func (c *Client) CallMethod(model string, method string, ids []int64, params map[string]interface{}) (interface{}, error) {
	client, err := xmlrpc.NewClient(c.ObjectURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create XML-RPC client: %w", err)
	}
	defer client.Close()

	args := []interface{}{
		c.Database,
		c.Uid,
		c.Password,
		model,
		method,
		[]interface{}{ids},
	}

	if params != nil {
		args = append(args, params)
	}

	var result interface{}
	if err := client.Call("execute_kw", args, &result); err != nil {
		return nil, fmt.Errorf("failed to call method %s: %w", method, err)
	}

	return result, nil
}

// Helper function to convert interface{} to specific types safely
func toInt64(v interface{}) (int64, bool) {
	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(val.Uint()), true
	case reflect.Float32, reflect.Float64:
		return int64(val.Float()), true
	}
	return 0, false
}
