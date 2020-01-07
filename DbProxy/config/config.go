// Copyright 2016 The kingshard Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package config

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

//用于通过api保存配置
var configFileName string

//整个config文件对应的结构
type Config struct {
	Addr     		string       	`yaml:"addr"`
	Redis RedisConfig `yaml:"redis"`

	PrometheusAddr 	string		 	`yaml:"prometheus_addr"`
	UserList 		[]UserConfig 	`yaml:"user_list"`

	WebAddr     string `yaml:"web_addr"`
	WebUser     string `yaml:"web_user"`
	WebPassword string `yaml:"web_password"`

	LogPath     string       `yaml:"log_path"`
	LogLevel    string       `yaml:"log_level"`
	LogSql      string       `yaml:"log_sql"`
	SlowLogTime int          `yaml:"slow_log_time"`
	AllowIps    string       `yaml:"allow_ips"`
	BlsFile     string       `yaml:"blacklist_sql_file"`
	Charset     string       `yaml:"proxy_charset"`
	Nodes       []NodeConfig `yaml:"nodes"`

	SchemaList []SchemaConfig `yaml:"schema_list"`

	ConfigVer  string          `yaml:"config_ver"`
}

//redis对应的配置
type RedisConfig struct {
	Host string `yaml:"host"`
	Port int `yaml:"port"`
	Db int `yaml:"db"`
	Password string `yaml:"password"`
}


//user_list对应的配置
type UserConfig struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

//node节点对应的配置
type NodeConfig struct {
	Name             string `yaml:"name"`
	DownAfterNoAlive int    `yaml:"down_after_noalive"`
	MaxConnNum       int    `yaml:"max_conns_limit"`

	User     string `yaml:"user"`
	Password string `yaml:"password"`

	Master string `yaml:"master"`
	Slave  string `yaml:"slave"`
}

//schema对应的结构体
type SchemaConfig struct {
	User      string        `yaml:"user"`
	Nodes     []string      `yaml:"nodes"`
	Default   string        `yaml:"default"` //default node
	ShardRule []ShardConfig `yaml:"shard"`   //route rule
	AllowDbList []string `yaml:"db_list"`   //allow db list
}

//range,hash or date
type ShardConfig struct {
	DB            string   `yaml:"db"`
	Table         string   `yaml:"table"`
	Key           string   `yaml:"key"`
	Nodes         []string `yaml:"nodes"`
	Locations     []int    `yaml:"locations"`
	Type          string   `yaml:"type"`
	TableRowLimit int      `yaml:"table_row_limit"`
	DateRange     []string `yaml:"date_range"`
}

//数据库节点
type ConfigDbNode struct {
	Id int `json:"id"`
	Host string `json:"host"`
	Port int    `json:"port"`
	UserName string   `json:"username"`
	Password string    `json:"password"`
}

//数据库用户
type ConfigDbUser struct {
	Id int `json:"id"`
	UserName string   `json:"username"`
	Password string    `json:"password"`
	NodeId int `json:"node_id"`
	AllowDbList []string `json:"db_list"`
}



func ParseConfigData(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	//redis连接读取配置
	redisAddr:=fmt.Sprintf("%s:%d",cfg.Redis.Host,cfg.Redis.Port)
	redisConn,redisConnErr:=redis.Dial("tcp",redisAddr,redis.DialDatabase(cfg.Redis.Db),redis.DialPassword(cfg.Redis.Password))
	if nil!=redisConnErr{
		fmt.Println(redisConnErr)
		return  nil,redisConnErr
	}

	defer  redisConn.Close()


	var(
		result string
		resultErr error
		jsonErr error
	)

	//数据版本
	result,resultErr=redis.String(redisConn.Do("get","cheer_dproxy_ver"))
	if nil!=resultErr{
		nowTime:=time.Now()
		result=nowTime.Format("2006-01-02 15:04:08")
	}

	cfg.ConfigVer=result


	//数据节点
	result,resultErr=redis.String(redisConn.Do("get","cheer_dproxy_nodes"))
	if nil!=resultErr{
		fmt.Println(resultErr)
		return &cfg,nil
	}

	var  dataConfigDbNodeList []ConfigDbNode
	jsonErr=json.Unmarshal([]byte(result),&dataConfigDbNodeList)
	if nil!=jsonErr{
		fmt.Println(jsonErr)
		return &cfg,nil
	}


	//数据用户
	result,resultErr=redis.String(redisConn.Do("get","cheer_dproxy_users"))
	if nil!=resultErr{
		fmt.Println(resultErr)
		return &cfg,nil
	}

	var dataConfigDbUserList []ConfigDbUser
	jsonErr=json.Unmarshal([]byte(result),&dataConfigDbUserList)
	if nil!=jsonErr{
		fmt.Println(jsonErr)
		return &cfg,nil
	}



	//添加节点
	for _,node:=range dataConfigDbNodeList{

		var tempNode NodeConfig
		tempNode.Name=fmt.Sprintf("dnode_%d",node.Id)
		tempNode.MaxConnNum=10
		tempNode.DownAfterNoAlive=10
		tempNode.User=node.UserName
		tempNode.Password=node.Password
		tempNode.Master=fmt.Sprintf("%s:%d",node.Host,node.Port)
		tempNode.Slave=fmt.Sprintf("%s:%d",node.Host,node.Port)

		cfg.Nodes=append(cfg.Nodes, tempNode)
	}


    //添加用户
	for _,user:=range dataConfigDbUserList{

		var tempUser UserConfig
		tempUser.User=user.UserName
		tempUser.Password=user.Password

		cfg.UserList=append(cfg.UserList,tempUser)


		nodeName:=fmt.Sprintf("dnode_%d",user.NodeId)

		var tempSchema SchemaConfig
		tempSchema.User=user.UserName
		tempSchema.Nodes= append(tempSchema.Nodes,nodeName )
		tempSchema.Default=nodeName
		tempSchema.AllowDbList=user.AllowDbList

		cfg.SchemaList=append(cfg.SchemaList,tempSchema)
	}

	return &cfg, nil
}

func ParseConfigFile(fileName string) (*Config, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	configFileName = fileName

	return ParseConfigData(data)
}

func WriteConfigFile(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(configFileName, data, 0755)
	if err != nil {
		return err
	}

	return nil
}
