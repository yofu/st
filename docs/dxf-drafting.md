# 製図スタイルと図枠（第②段階）

寸法付き伏図・軸組図と図面一式に共通の製図スタイルを適用します。
既存の `.drawing.json` でも `style` / `sheet` がなければ標準値を使います。
`:drawinginit` で新規作成する設定には標準スタイルと図枠設定を含めます。
解析モデルの座標・断面・荷重は変更しません。

## 標準値

| 対象 | 線幅（用紙上mm） | 線種 |
|---|---:|---|
| 柱 | 0.35 | 実線 |
| 梁 | 0.25 | 実線 |
| ブレース | 0.13 | 破線 |
| 通り芯・階レベル | 0.13 | 一点鎖線 |
| 寸法・文字・引出線 | 0.13 | 実線 |
| 断面外形・図枠 | 0.25 | 実線 |
| 鉄筋 | 0.13 | 実線 |

白黒出力、フォント `msgothic.ttc`、部材符号・通り名・注記・寸法文字の
標準高さ2.5 mm、文字幅係数0.9です。寸法矢印1.5 mm、表示小数桁0です。
通り芯の円は実線に保ちます。鋼材寸法の浮動小数の桁あふれも抑制します。

線幅はDXFの規定値へ切り上げます（例: 0.26 → 0.30 mm）。
`monochrome: false` にすると元のレイヤー色を使い、必要な役割だけ `color`
（ACI 1～255）で上書きできます。白黒指定は色指定より優先します。
CTBを必要とせず、レイヤー線幅を印刷するページ設定を出力します。
CAD側には参照フォントが必要です。

## 設定例

初期設定の他の項目を残し、次の項目を追加・編集します。

```json
{
  "version": 1,
  "style": {
    "monochrome": true,
    "font": "msgothic.ttc",
    "text_height_mm": 2.5,
    "text_width_factor": 0.9,
    "grid_text_height_mm": 2.5,
    "note_text_height_mm": 2.5,
    "dimension": {
      "text_height_mm": 2.5,
      "arrow_size_mm": 1.5,
      "text_gap_mm": 0.75,
      "extension_offset_mm": 1.0,
      "extension_extend_mm": 1.5,
      "decimals": 0
    },
    "layers": {
      "column": {"line_weight_mm": 0.35, "line_type": "continuous"},
      "beam": {"line_weight_mm": 0.25, "line_type": "continuous"},
      "brace": {"line_weight_mm": 0.13, "line_type": "dashed"},
      "grid": {"line_weight_mm": 0.13, "line_type": "center"}
    }
  },
  "sheet": {
    "layout": "right",
    "paper": "A3",
    "margin_mm": 10,
    "title_width_mm": 45,
    "office": "構造設計事務所名",
    "structure": "鉄骨造",
    "project": "工事名称",
    "date": "2026-11-21",
    "drawn_by": "作図担当者",
    "checked_by": "",
    "number_prefix": "S-",
    "start_number": 7,
    "number_digits": 2,
    "notes": ["寸法単位：mm"],
    "revisions": [
      {"mark": "A", "date": "2026-11-21", "description": "改訂内容", "by": "担当者"}
    ],
    "view_rotations": {"elevation:Y2": 90, "elevation:X1": 90},
    "pages": {"1": {"title": "3階伏図", "number": "S-07"}}
  }
}
```

大きさの省略または `0` は標準値を使用します。
`line_weight_mm` と `decimals` は明示的な `0` を指定できます。
役割は `column` / `beam` / `brace` / `grid` / `floor` / `dimension` /
`text` / `section` / `rebar` / `frame` / `leader` です。

## 図枠と配置

- 右側タイトル欄の領域を先に確保し、残りへ図を配置して自動縮尺を選び直します。
- 図名は配置された図から、縮尺は実際の縮尺から生成します。
  混在時は各縮尺と「各図参照」を表示します。
- 図番は開始番号から連番です。`pages` でページごとの図名・図番を指定できます。
  重複する図番はエラーにします。
- 担当者・日付は設定値を表示し、自動では埋めません。
- 長い文字は折り返し、収まらなければエラーを返します。
  断面カードも折り返し、ビューポートで文字が切れる出力を防ぎます。
- 簡易な下側欄は `layout: "bottom"` です。
  事務所・構造種別・共通注記・改訂欄は右側テンプレートを使用してください。
- 実部材の位置・開口・床仕様等の追加は③以降です。

## 回転と縮尺

`view_rotations` のキーは `plan:3` / `elevation:Y2` などです。
角度は0 / 90 / 180 / 270度で、回転後の幅・高さで用紙への収まりを確認します。
手動移動量は回転・文字寸法を変えても用紙上のXY方向で保ちます。

```text
:drawings sheets.dxf -plans=3 -grids=Y2,X1 -sections
:drawings axes.dxf -grids=Y2,X1 -rotate=90 -scale=100
```

`-rotate` はそのコマンドの全ビューを上書きします。
MCPでは各ビューの `rotation` を個別に指定できます。
`-paper` / `-font` / `-title` は設定ファイルより優先します。

線種の基本長さは用紙上で、破線3 mm＋空白1.5 mm、
一点鎖線8 mm＋空白1.5 mm＋短線0.5 mm＋空白1.5 mmです。
単独の寸法付きDXFでは出力縮尺に換算し、
図面一式では `PSLTSCALE=1` と各ビューポートで紙上寸法を揃えます。

## MCPとレポート

①と同じ `query_dxf_settings` → `preview_dxf_settings` → `save_dxf_settings`
でスタイル・図枠も保存できます。`preview_dxf_drawings` →
`save_dxf_drawings` で再出力します。
レポートの `drafting` / `sheet_template` は解決後の設定、
`sheets` は図番・図名・実縮尺・作図領域・タイトル欄の領域です。
