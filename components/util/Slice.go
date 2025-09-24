package util

func AddToSlice(columnName string, condition string, operator string, value interface{},
	columnNames *[]string, conditions *[]string, operators *[]string, values *[]interface{}) {
	if len(*columnNames) != 0 {
		*operators = append(*operators, operator)
	}
	*columnNames = append(*columnNames, columnName)
	*conditions = append(*conditions, condition)
	*values = append(*values, value)
}
