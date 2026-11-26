package model

// BuildRule 是一次构建的「引用规则」，由环境配置派生。
// 过去这几件事散落在 NormalizeRef / resolveParamName / TriggerBuild 三处各自判断 isProd，
// 现在统一从这里取值，避免同一个开关被解释出不同含义。
type BuildRule struct {
	ParamName  string // 传给 Jenkins 的参数名
	DefaultRef string // 分支模式下留空时回落的引用
	RequireRef bool   // Tag 模式必须显式给值，不允许回落
}

// Rule 从环境配置导出构建规则。ParamName 为空时按模式给内置默认。
func (e Environment) Rule() BuildRule {
	if e.IsProd {
		name := e.ParamName
		if name == "" {
			name = "Tags"
		}
		return BuildRule{ParamName: name, RequireRef: true}
	}

	name := e.ParamName
	if name == "" {
		name = "branch"
	}
	return BuildRule{
		ParamName:  name,
		DefaultRef: e.DefaultRefValue(),
		RequireRef: false,
	}
}

// SyncParamName 把 API 契约里的两个参数名（分支/Tag）收敛到单个存储字段。
// 表单始终提交两个值，但只有当前模式对应的那个会生效，另一个不再落库成为死数据。
func (e *Environment) SyncParamName() {
	if e.IsProd {
		e.ParamName = e.DefaultTagParam
	} else {
		e.ParamName = e.DefaultBranchParam
	}
}

// ExpandParamName 读取时把单字段还原成 API 契约的两个字段，保证前端无感知。
func (e *Environment) ExpandParamName() {
	if e.ParamName == "" {
		return
	}
	if e.IsProd {
		e.DefaultTagParam = e.ParamName
		e.DefaultBranchParam = ""
	} else {
		e.DefaultBranchParam = e.ParamName
		e.DefaultTagParam = ""
	}
}
