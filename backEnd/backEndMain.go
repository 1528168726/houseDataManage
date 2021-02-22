package backEnd

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type dataBaseInfo struct {
	userName string
	password string
	ip       string
	port     string
	dbName   string
}

const SearchLenLimit = 7

var DB *sql.DB

type HousingResource struct {
	//id只在读取时有用
	HouseId int
	//isSell代表是否为出售，否则为出租
	IsSell bool
	//楼盘地址
	EstateAddr string
	//楼栋号
	BuildingAddr int
	//详细地址，格式：单元号-楼层-门牌
	DetailAddr string
	//户型
	HouseType string
	//面积
	Area float64
	//售价 万元
	Price float64
	//朝向
	Orientation string
	//房主
	Owner TypeOwner
	//录入者
	Inputer string
	//录入时间，格式年-月-日
	Date string
	//备注
	Remark string
	//输出用
	UnitPrice float64
	Floor     string
}

type TypeOwner struct {
	Name         string
	PhoneNumber  string
	WeChatNumber string
}

type TypeListOrder int

const (
	D_date = iota
	I_date
	I_estateAddr
	D_estateAddr
	I_area
	D_area
	I_price
	D_price
	I_unitPrice
	D_unitPrice
)

type SearchKey struct {
	EstateAddr   string
	BuildingAddr int
	Begin        int
	ListOrder    TypeListOrder
}

type rowScan interface {
	Scan(dest ...interface{}) error
}

func init() {
	var err error
	var fPath string
	if runtime.GOOS == "linux" {
		fPath = "dataBaseInfo.txt"
	} else if runtime.GOOS == "windows" {
		fPath = "dataBaseInfo.txt"
	}

	f, err := os.Open(fPath)
	defer f.Close()
	check(err)
	var d dataBaseInfo

	_, err = fmt.Fscan(f, &d.userName, &d.password, &d.ip, &d.port, &d.dbName)
	check(err)

	path := strings.Join([]string{d.userName, ":", d.password, "@tcp(", d.ip, ":", d.port, ")/", d.dbName, "?charset=utf8"}, "")
	DB, err = sql.Open("mysql", path)
	check(err)
	err = DB.Ping()
	check(err)
	DB.SetMaxOpenConns(100)
	DB.SetMaxIdleConns(10)
	//println("ok")
	//defer DB.Close()
}

func InsertHousingResource(data *HousingResource) error {
	data.checkHousingResource()
	sqlStr := `insert into tbHouseData  
(isSell,estateAddr,buildingAddr,detailAddr,houseType,area,price,
 orientation,ownerName,ownerPhoneNumber,weChatNumber,inputer,date,remark)
values
(?,?,?,?,?,?,?,?,?,?,?,?,?,?);
`
	_, err := DB.Exec(sqlStr, data.IsSell, data.EstateAddr, data.BuildingAddr,
		data.DetailAddr, data.HouseType, data.Area, data.Price, data.Orientation, data.Owner.Name,
		data.Owner.PhoneNumber, data.Owner.WeChatNumber, data.Inputer, data.Date, data.Remark)
	if err != nil {
		return err
	}
	return nil
}

func (h *HousingResource) checkHousingResource() {
	if h.Date == "" {
		t := time.Now()
		h.Date = t.String()[0:10]
		//println(h.Date)
	}
	if h.Area <= 0 {
		h.Area = 1
	}
}

func DeleteHouseResource(houseId int) error {
	sqlStr := `insert into tbHouseDataBackUp select *from tbHouseData where houseId = ?;`
	_, err := DB.Exec(sqlStr, houseId)
	if err != nil {
		return err
	}

	sqlStr = `delete from tbHouseData where houseId = ?;`
	re, err := DB.Exec(sqlStr, houseId)
	if err != nil {
		return err
	}
	if i, _ := re.RowsAffected(); i != 1 {
		return errors.New("delete house resource fail, no such id")
	}
	return nil
}

func GetAllEstateAddr() (*[]string, error) {
	sqlStr := `select distinct estateAddr from tbHouseData;`
	re, err := DB.Query(sqlStr)
	if err != nil {
		return nil, err
	}
	defer re.Close()
	var result []string
	for re.Next() {
		var tmp string
		err = re.Scan(&tmp)
		result = append(result, tmp)
		if err != nil {
			return nil, err
		}
	}
	return &result, nil
}

//return housingResources and the nums of all count
func SearchData(key *SearchKey) (*[]HousingResource, int, error) {
	sqlStr := `select *from tbHouseData where estateAddr like "%` + key.EstateAddr + `%" `
	sqlStrCount := `select count(houseId)  from tbHouseData where estateAddr like "%` + key.EstateAddr + `%" `
	if key.BuildingAddr != 0 {
		sqlStr += ` And buildingAddr =` + strconv.Itoa(key.BuildingAddr) + ` `
		sqlStrCount += ` And buildingAddr =` + strconv.Itoa(key.BuildingAddr) + ` `
	}
	row := DB.QueryRow(sqlStrCount)
	var count int
	row.Scan(&count)

	switch key.ListOrder {
	case I_estateAddr:
		sqlStr += `order by estateAddr ASC`
	case D_estateAddr:
		sqlStr += `order by estateAddr DESC`
	case I_area:
		sqlStr += `order by area ASC`
	case D_area:
		sqlStr += `order by area DESC`
	case I_date:
		sqlStr += `order by date ASC`
	case D_date:
		sqlStr += `order by date DESC`
	case I_price:
		sqlStr += `order by price ASC`
	case D_price:
		sqlStr += `order by price DESC`
	case I_unitPrice:
		sqlStr += `order by price/area ASC`
	case D_unitPrice:
		sqlStr += `order by price/area DESC`
	}
	sqlStr += ` limit ?,?;`
	re, err := DB.Query(sqlStr, key.Begin, SearchLenLimit)
	if err != nil {
		return nil, 0, err
	}

	var result []HousingResource
	for re.Next() {
		tmp, err := scanHousingResource(re)
		if err != nil {
			return nil, 0, err
		}
		fillData(tmp)
		result = append(result, *tmp)
	}
	return &result, count, nil
}

func GetData(houseId int) (*HousingResource, error) {
	sqlStr := `select *from tbHouseData where houseId=?;`
	row := DB.QueryRow(sqlStr, houseId)
	data, err := scanHousingResource(row)
	if err != nil {
		return nil, err
	}
	fillData(data)
	return data, nil
}

func ChangeData(houseId int, data *HousingResource) error {
	err := DeleteHouseResource(houseId)
	if err != nil {
		return err
	}
	err = InsertHousingResource(data)
	if err != nil {
		return err
	}
	return nil
}

func IsRepeat(data *HousingResource) bool {
	sqlStr := `select count(houseId)  from tbHouseData where TRIM(estateAddr) = TRIM(?) and buildingAddr=? and TRIM(detailAddr) =TRIM(?)`
	res := DB.QueryRow(sqlStr, data.EstateAddr, data.BuildingAddr, data.DetailAddr)
	n := 0
	res.Scan(&n)
	if n > 0 {
		return true
	}
	return false

}

func scanHousingResource(re rowScan) (*HousingResource, error) {
	tmp := new(HousingResource)
	err := re.Scan(&tmp.HouseId, &tmp.IsSell, &tmp.EstateAddr, &tmp.BuildingAddr, &tmp.DetailAddr,
		&tmp.HouseType, &tmp.Area, &tmp.Price, &tmp.Orientation, &tmp.Owner.Name,
		&tmp.Owner.PhoneNumber, &tmp.Owner.WeChatNumber, &tmp.Inputer,
		&tmp.Date, &tmp.Remark)
	return tmp, err
}

func fillData(data *HousingResource) {
	var a string
	fmt.Sscan(data.DetailAddr, &a, &data.Floor)
	data.UnitPrice = data.Price / data.Area
	data.UnitPrice = float64(int(data.UnitPrice*100)) / 100
}

//func Test()  {
//	a:="tmp"
//	b:=`select * from ?;`
//	t,err:=DB.Query(b,a)
//	if err != nil {
//		panic(err)
//	}
//	for t.Next() {
//		third :=-1
//		t.Scan(&third)
//		println(third)
//	}
//}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
