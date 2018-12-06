package cmd

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var Cmd = cmd()

func init() {
	cobra.OnInitialize(initConfig)

	Cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.config.yml)")

	viper.BindPFlag("url", Cmd.PersistentFlags().Lookup("url"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".config")
	viper.AddConfigPath("$HOME")
	viper.AutomaticEnv()

	viper.ReadInConfig()
}

type Options struct {
	database string
	table    string
}

var (
	o = &Options{}
)

func cmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:           "make graphql field file",
		Short:         "make graphql field file",
		RunE:          cmdFunction,
		SilenceErrors: true,
		SilenceUsage:  false,
	}
	cmd.Flags().StringVarP(&o.database, "database", "d", "", "database source (ex:root:password@tcp(127.0.0.1:3306)/database?charset=utf8mb4&parseTime=True&loc=Asia%2FTokyo) [required]")
	cmd.Flags().StringVarP(&o.table, "table", "t", "", "database table [required]")

	cmd.MarkFlagRequired("database")
	cmd.MarkFlagRequired("table")

	return cmd
}

type ColumnInfo struct {
	ColumnName    string
	ColumnType    string
	ColumnComment string
}

func cmdFunction(cmd *cobra.Command, args []string) error {
	c := newCmdClient(o.database)

	cis, err := c.GetColumnInfo(o.table)
	if err != nil {
		return err
	}

	s := c.GetGraphQLType(o.table, cis)
	fmt.Println(s)

	cmd.OutOrStdout()

	return nil
}

type cmdClient struct {
	db *gorm.DB
}

func newCmdClient(database string) *cmdClient {
	db, err := gorm.Open("mysql", fmt.Sprintf("%s", database))
	if err != nil {
		log.Fatalf("err : %v", err)
	}
	db.LogMode(true)

	return &cmdClient{
		db: db,
	}
}

func (c *cmdClient) GetColumnInfo(table string) ([]ColumnInfo, error) {
	cis := []ColumnInfo{}

	if re := c.db.
		Table("information_schema.columns").
		Select([]string{"column_name", "column_type", "column_comment"}).
		Where("table_name = ?", table).
		Order("ordinal_position").
		Find(&cis); re.Error != nil {
		return nil, re.Error
	}

	return cis, nil
}

func (c *cmdClient) GetGraphQLType(table string, cis []ColumnInfo) string {

	camelTable := snakeToCamel(table)
	name := fmt.Sprintf("%s%s", strings.ToUpper(camelTable[0:1]), camelTable[1:len([]rune(camelTable))-1])
	str := fmt.Sprintf("var %sType = graphql.NewObject(graphql.ObjectConfig{\n", name)
	str = str + fmt.Sprintf("	Name: \"%s\",\n", name)
	str = str + fmt.Sprintf("	Fields: graphql.Fields{\n")
	for _, ci := range cis {
		str = str + fmt.Sprintf("		\"%s\": &graphql.Field{Type: graphql.%s, Description: \"%s\"},\n", snakeToCamel(ci.ColumnName), columnType(ci.ColumnName, ci.ColumnType), ci.ColumnComment)
	}
	str = str + fmt.Sprintf("	},\n")
	str = str + fmt.Sprintf("})\n")

	return str
}

var snakeRegex = regexp.MustCompile("_([a-z])")

func snakeToCamel(s string) string {
	return snakeRegex.ReplaceAllStringFunc(s, func(m string) string {
		return strings.ToUpper(m[1:])
	})
}

func columnType(cName, cType string) string {
	switch {
	case cName == "id":
		return "ID"
	case strings.HasPrefix(cType, "varchar"):
		return "String"
	case strings.HasPrefix(cType, "char"):
		return "String"
	case strings.HasPrefix(cType, "int"):
		return "Int"
	case strings.HasPrefix(cType, "tinyint"):
		return "Int"
	case strings.HasPrefix(cType, "float"):
		return "Float"
	case strings.HasPrefix(cType, "time"):
		return "DateTime"
	case strings.HasPrefix(cType, "date"):
		return "DateTime"
	}

	return ""
}
