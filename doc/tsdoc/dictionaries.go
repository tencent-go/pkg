package tsdoc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"text/template"

	"github.com/tencent-go/pkg/doc/schema"
	"github.com/tencent-go/pkg/util"
	"github.com/sirupsen/logrus"
)

func NewDictionariesFile(schemaPackages []*schema.Package, parentDir ...string) util.DataFile {
	var dictKeysTmp = `
export type Locale = '' | 'zh-CN' | 'zh-TW' | 'zh-HK' | 'en-US' | 'en-GB' | 'en-AU' | 'en-CA' | 'en-IN' | 'fr-FR' | 'fr-CA' | 'de-DE' | 'fil-PH' | 'de-CH' | 'es-ES' | 'es-MX' | 'es-US' | 'ja-JP' | 'ko-KR' | 'ru-RU' | 'pt-BR' | 'pt-PT' | 'ar-SA' | 'ar-EG' | 'hi-IN' | 'it-IT' | 'it-CH' | 'nl-NL' | 'nl-BE' | 'pl-PL' | 'vi-VN' | 'th-TH' | 'el-GR' | 'tr-TR' | 'sv-SE';

export type ValueType = string | number | undefined;

export type DefaultLocalizedExtra = { tip?: string };

export type LocalizedItem<T extends object = DefaultLocalizedExtra> = Partial<T> & {
  label: string;
}

type Localized<T extends object = DefaultLocalizedExtra> = { '': LocalizedItem<T> } & Partial<Omit<Record<Locale, LocalizedItem<T>>, ''>>

type OptionItem<T extends ValueType = string, U extends object = DefaultLocalizedExtra, V extends object = {}> = Partial<V> & {
  value: T;
  localized: Localized<U>;
}

export type DisplayOption<T extends ValueType, U extends object = DefaultLocalizedExtra, V extends object = {}> = Partial<V> & {
  value: T;
} & LocalizedItem<U>;

export class OptionRegistry<T extends ValueType = string, U extends object = DefaultLocalizedExtra, V extends object = {}> {
  private readonly items: OptionItem<T, U, V>[];
  private readonly byValue: Map<T, OptionItem<T, U, V>>;

  constructor(items: OptionItem<T, U, V>[]) {
    this.items = items;
    this.byValue = new Map(this.items.map(item => [item.value, item]));
  }

  getValues(): T[] {
    return this.items.map(item => item.value);
  }

  size(): number {
    return this.items.length;
  }

  toMap(locale: Locale = ''): Map<T, DisplayOption<T, U, V>> {
    return new Map(this.items.map(item => [item.value, this.getOption(item.value, locale)]));
  }

  toArray(locale: Locale = ''): DisplayOption<T, U, V>[] {
    return this.items.map(({ value }) => this.getOption(value, locale));
  }

  getOption(value: T, locale: Locale = ''): DisplayOption<T, U, V> {
    const item = this.byValue.get(value);
    if (!item) {
      return { value: value, label: value?.toString() ?? '' } as DisplayOption<T, U, V>;
    }
    const { localized, ...extra } = item;
    const l = localized[locale] || localized['']!;
    return { ...l, ...extra } as DisplayOption<T, U, V>;
  }

  setLocalizedExtra(locale: Locale, params: ObjectParams<T, LocalizedItem<U>>) {
    Object.entries(params).forEach(([key, extra]) => {
      const k = (key === '_undefined' ? undefined : key) as T;
      const item = this.byValue.get(k);
      if (item) {
        item.localized[locale] = extra as LocalizedItem<U>;
      } else {
        console.error("Option "+key+" not found in OptionRegistry ["+this.getValues().join(', ')+"]");
      }
    });
    return this;
  }

  withLocalizedExtra<E extends object = {}>(locale: Locale, values: ObjectParams<T, LocalizedItem<U & E>>) {
    return new OptionRegistry<T, U & E, V>(this.items as OptionItem<T, U & E, V>[]).setLocalizedExtra(locale, values);
  }

  setExtra(params: ObjectParams<T, V>) {
    Object.entries(params).forEach(([key, extra]) => {
      const k = (key === '_undefined' ? undefined : key) as T;
      const item = this.byValue.get(k);
      if (item) {
        this.byValue.set(k, { ...item, ...extra as any } as OptionItem<T, U, V>);
      } else {
        console.error("Option "+key+" not found in OptionRegistry ["+this.getValues().join(', ')+"]");
      }
    });
  }

  withExtra<E extends object = {}>(values: ObjectParams<T, V & E>) {
    return new OptionRegistry<T, U, V & E>(this.items as OptionItem<T, U, V & E>[]).setExtra(values);
  }

  union<E extends ValueType>(other: OptionRegistry<E, U, V>): OptionRegistry<T | E, U, V> {
    return new OptionRegistry([...this.items, ...other.items] as OptionItem<T | E, U, V>[]);
  }

  excludes<K extends T>(...values: readonly K[]): OptionRegistry<Exclude<T, K>, U, V> {
    return new OptionRegistry(this.items.filter(item => !values.includes(item.value as K)) as OptionItem<Exclude<T, K>, U, V>[]);
  }

}

export function createOptionRegistry<T extends ValueType = string>(...items: DisplayOption<T>[]): OptionRegistry<T> {
  return new OptionRegistry(items.map(item => ({
    value: item.value,
    localized: {
      '': {
        label: item.label,
        tip: item.tip,
      }
    }
  })));
}

export function option<T extends ValueType = string>(value: T, label?: string): DisplayOption<T> {
  return {
    value,
    label: label ?? value?.toString() ?? ''
  }
}

type ObjectParams<T extends ValueType, U> = Partial<{
  [K in T as K extends undefined ? '_undefined' : K]: U;
}>

const optionRegistries = {
{{range .}}  '{{.Path}}': new OptionRegistry<{{.Path}}>({{.Values}}),
{{end}}}

export default optionRegistries;
`

	t, err := template.New("typescript_dict_keys").Parse(dictKeysTmp)
	if err != nil {
		logrus.Fatalf("parse dict keys template failed: %v", err)
	}
	f := util.DataFile{
		Name: "options.ts",
		Dir:  path.Join(parentDir...),
	}
	var data []registry
	for _, pkg := range schemaPackages {
		if len(pkg.Enums) == 0 {
			continue
		}
		for _, c := range pkg.Enums {
			items := make([]item, len(c.Items))
			for idx, v := range c.Items {
				it := item{
					Value:     v.Value,
					Localized: make(map[string]localizedItem),
				}
				for locale, info := range v.LocalizedInfo {
					k := string(locale)
					it.Localized[k] = localizedItem{Label: info.Label}
				}
				if _, hasDef := it.Localized[""]; !hasDef {
					it.Localized[""] = localizedItem{Label: fmt.Sprintf("%v", v.Value)}
				}
				items[idx] = it
			}
			values, e := json.Marshal(items)
			if e != nil {
				logrus.Fatalf("marshal enum items failed: %v", e)
			}
			strValues := strings.ReplaceAll(string(values), "\"", "'")
			data = append(data, registry{
				Path:   fmt.Sprintf("%s.%s", pkg.Name, c.Name),
				Values: strValues,
			})
		}
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, data); err != nil {
		logrus.Fatalf("execute dict keys template failed: %v", err)
	}
	f.Data = buf.Bytes()
	return f
}

type registry struct {
	Path   string
	Values string
}

type item struct {
	Value     any                      `json:"value"`
	Localized map[string]localizedItem `json:"localized"`
}

type localizedItem struct {
	Label string `json:"label"`
}
