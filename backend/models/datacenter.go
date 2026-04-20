package models

import "time"

type Datacenter struct {
	ID             uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Name           string    `json:"name" gorm:"uniqueIndex;size:200;not null"`
	Remark         string    `json:"remark"`
	Campus         string    `json:"campus" gorm:"size:200"`                        // 所属数据中心/园区
	Location       string    `json:"location" gorm:"size:300"`                      // 地理位置
	Floor          string    `json:"floor" gorm:"size:50"`                          // 楼层
	Room           string    `json:"room" gorm:"size:50"`                           // 房间号
	Contact        string    `json:"contact" gorm:"size:100"`                       // 联系人
	OperationMode  string    `json:"operation_mode" gorm:"size:50"`                 // 运营方式: 自建/托管/租赁
	CurrentStatus  string    `json:"current_status" gorm:"size:50;default:'运行中'"`  // 当前状态: 运行中/建设中/停用
	MaxU           int       `json:"max_u" gorm:"default:47"`                       // 机柜最大U数
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CabinetColumn struct {
	ID           uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	DatacenterID uint   `json:"datacenter_id" gorm:"index;not null"`
	Name         string `json:"name" gorm:"size:100;not null"`
	SortOrder    int    `json:"sort_order"`
	ColumnType   string `json:"column_type" gorm:"size:50;default:'cabinet'"` // cabinet/hda/pdu/aircon/other
}

type CabinetRow struct {
	ID           uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	DatacenterID uint   `json:"datacenter_id" gorm:"index;not null"`
	Name         string `json:"name" gorm:"size:100"`
	SortOrder    int    `json:"sort_order"`
}

type Cabinet struct {
	ID           uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	DatacenterID uint   `json:"datacenter_id" gorm:"index;not null"`
	ColumnID     *uint  `json:"column_id"`
	RowID        *uint  `json:"row_id"`
	Name         string `json:"name" gorm:"size:100;not null"`
	Height       int    `json:"height" gorm:"default:47"`  // 机柜高度U数
	Width        int    `json:"width" gorm:"default:60"`   // 宽cm
	Depth        int    `json:"depth" gorm:"default:120"`  // 深cm
	CabinetType  string `json:"cabinet_type" gorm:"size:50;default:'standard'"` // standard/hda/pdu/aircon
	Remark       string `json:"remark"`
}
