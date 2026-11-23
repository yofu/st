# stmcp

STの既存モデルを操作するローカルMCPサーバーです。MCPクライアントが標準入出力で起動します。

## ビルドと接続

リポジトリのルートから実行します。

```powershell
go build -o stmcp.exe ./cmd/stmcp
```

MCPクライアントのサーバー設定で、コマンドに `stmcp.exe` の絶対パス、引数に `-root` とモデルのあるディレクトリを指定します。

```json
{
  "command": "C:/tools/stmcp.exe",
  "args": ["-root", "C:/models"]
}
```

各ツールのファイルパスは `-root` からの相対パスです。接続ごとに編集中のモデルを1件保持します。同名のLSTがあれば、INPと一緒に読み込みます。

## 編集手順

1. `open_model` に `{"path":"frame.inp"}` を渡します。
2. `query_nodes`、`query_elements`、`query_sections` で対象を調べます。個別照会は `{"ids":[102]}`、一覧は `{"limit":50}` で取得し、続きは返り値の `next_after` を `after` に指定します。
3. `preview_edits` に変更を渡すと、各項目の変更前後と `proposal_id` が返ります。断面変更は、部材にモデル内の既存断面を割り当てます。拘束・荷重は指定した6成分の配列全体を置き換えます。

   ```json
   {
     "edits": [
       {"element": 1002, "section": 211},
       {"node": 102, "fixed": [true, true, true, false, false, true],
        "load": [0, 0, -1.25, 0, 0, 0.125]}
     ]
   }
   ```

4. 差分を確認後、`save_model_as` に `{"proposal_id":1,"path":"frame-edited.inp"}` を渡します。既存ファイルへの上書きはエラーになります。保存したモデルを照会するには、新しいファイルを `open_model` で開きます。

節点拘束と荷重の順序は X、Y、Z、RX、RY、RZ です。座標は m、荷重は元のINPと同じ tf・tf·m です。変更案は `stlib` のパーサーで再読み込みし、指定値が保持されることを確認します。出力は元ファイルの指定箇所だけを書き換えるため、コメントや対象外の数値精度も保持されます。

## 解析と変更前後の比較

現在のモデルと `preview_edits` の変更案を比較します。変更案は保存せずに解析できます。次の呼び出しは、長期荷重Lの線形静的解析と固有振動解析を、両モデルに対して起動します。

```json
{"proposal_id":1,"analysis":"both","period":"L","node_ids":[102,131],"modes":3}
```

1. 上記の引数を `start_analysis_comparison` に渡し、`job_id` を取得します。`analysis` は `static`、`vibration`、`both`、`period` は `L`、`X`、`Y` を指定できます。静的解析を含む場合は照会対象の `node_ids` が必要です。
2. `analysis_status` に `{"job_id":1}` のように渡すと、前後どちらのモデルか、処理段階、ソルバーの進捗、経過秒数を確認できます。停止は同じ引数で `cancel_analysis` を呼びます。
3. 完了後、`analysis_results` で変更前・変更後・差（変更後−変更前）を取得します。

変位は m・rad、反力は tf・tf·m、周期は秒です。節点の成分順は X、Y、Z、RX、RY、RZ です。周期は各モデルの振動数の昇順で比較します。モード形状による対応付けは行いません。振動は指定した荷重ケースの抽出モデルを使う初期剛性の固有値解析です。

解析は別プロセスで実行します。1接続につき1ジョブを実行でき、次のジョブを起動するまで最新ジョブの結果を取得できます。停止時は途中結果を破棄します。重量・解析の中間ファイルは専用の一時フォルダに出力されます。

## 座屈と材料非線形解析の比較

同じ `start_analysis_comparison` に、確認済みの `proposal_id` と次の引数を渡します。

| 解析 | 呼び出し例 |
|---|---|
| 座屈固有値解析 | `{"proposal_id":1,"analysis":"buckling","period":"L","modes":3}` |
| 材料非線形静的解析 | `{"proposal_id":1,"analysis":"arclm101","period":"X","preload_l":true,"node_ids":[131],"steps":4,"increment":0.05}` |

- `buckling` は `Bclng001` で指定した荷重ケースの正の座屈荷重係数を求め、昇順のモード番号ごとに前後と差を返します。
- `arclm101` は材料非線形を考慮し、収束した増分ごとに荷重係数と指定節点の6成分の変位を前後2系列で返します。変位配列の節点順は `node_ids` と同じです。変位は m・rad、荷重係数は指定ケースの無次元倍率です。
- `preload_l:true` は `X` または `Y` の解析前に線形静的解析で `L` を載荷します。たとえば荷重係数 0.20 は `L+0.20X` を表します。出力の変位には先行載荷分も含まれます。省略時は指定ケースをゼロ荷重から載荷します。
- 後続増分が収束しなければ、最後の収束点までの曲線と `nonconvergence` を該当系列に返します。これはソルバーの収束限界です。途中停止では途中結果を破棄します。

進捗取得・停止・結果取得は、線形解析と同じツールを使います。材料非線形解析の進捗には、最後に収束したステップ番号、指定ステップ数、荷重係数が `step_progress` に入ります。

## GUIのモデル編集

`list_model_editors` で利用できる組み込みコマンドを確認できます。`preview_gui_edits` は選択と操作を順番に再現し、変更前後のINPレコードを返します。取得した `proposal_id` は `save_model_as` と `start_analysis_comparison` にそのまま渡せます。途中でエラーになった場合は、バッチ全体を破棄します。

```json
{
  "commands": [
    {"command": ":move 0 0 0.025", "node_ids": [102], "element_ids": []},
    {"command": "BONDPIN", "node_ids": [], "element_ids": [1201]},
    {
      "command": "KIJUN",
      "inputs": [
        {"kind": "node", "id": 102},
        {"kind": "node", "id": 131},
        {"kind": "text", "text": "通り A"}
      ]
    }
  ]
}
```

- `node_ids`、`element_ids` は指定順に選択します。省略すると直前の操作から引き継ぎ、空配列で選択を解除します。ダイアログの確認は `confirm:true` で承認します。
- 対話コマンドには、節点・部材を指す `node` / `element`、入力欄の `text`、キーボードの `key`、右クリックに相当する `finish` を順に渡します。座標ベースの操作には `click` / `fence` を使います。例えば `{"kind":"fence","xy":[100,200],"end":[300,200]}` の座標はモデルのGio投影座標です。`point` / `end_point` にモデルのX・Y・Z座標を渡すと、サーバーで投影します。
- exの引数はコマンド文字列に指定します。パイプの各段に `:` を付けます。新規座標は `:addnode X Y Z`、選択節点からの部材作成は `:add elem -sect=101 -etype=GIRDER`、節点荷重は `:pload 2 -1.25` です。
- IUPの断面ダイアログに対応する操作は `:add sect 101`、`ADD_SECTION_FIG 101 1`（断面番号、材料番号）、`EDIT_SECTION_FIG 101 1 AREA=0.02 IXX=0.0001 IYY=0.0002 VEN=0.00005`、`SECTION_COLOR 101 70 90 120`、`DELETE_SECTION 101` です。板要素の開口は選択した部材に `EDITWRECT 0.8 1.2` で指定します。
- ファイルの挿入・断面の取り込みは `:insert relative.inp 0`（基点節点を選択）、`:propsect relative.inp` を使います。ファイル名はサーバーのルートから指定します。`:read kijun relative.kjn` と `:read weight relative.wgt` も使えます。`:read result relative.otl` は同じバッチ内で `:camber` などに使う解析結果を読み込みます。
- 保留中の提案に追記する場合は、`preview_gui_edits` に `proposal_id` を渡します。`UNDO` / `REDO` は同じバッチ内で使えます。保存した基準線・寸法線・円弧・編集ロックは、新しいSTで開くと復元されます。

### テンセグリティ

`TENSEGRITYADD 圧縮断面 引張断面 各端の候補数` は圧縮材を選択して使います。`TENSEGRITYCONNECTED 圧縮断面 引張断面` は接続している圧縮材の番号を返します。

`TENSEGRITYDELETE 圧縮断面 引張断面 残す本数` は削除候補を計算する前に、同じバッチで `:read result model.otl`、`:read result model.ohx`、`:read result model.ohy` を実行してください。最後のコマンドに `node_ids` を指定し、`confirm` を省略すると候補の部材番号だけを確認できます。`confirm:true` 付きで再実行すると削除の提案になります。候補材の軸力が欠けている場合はエラーを返し、同じ軸力評価値の候補は部材番号順で扱います。

## MCPで長期荷重の静的解析を実行し、IUPに表示する

更新した `st.exe` と `stmcp.exe` を使います。IUPで `:mcpconnect` を入力し、履歴に出る接続IDを使って次の順でツールを呼び出します。

1. `open_model` に `{"path":"nakano02_07.inp","connection_id":"接続ID"}` を渡します。MCPとIUPの双方で同じINPを開きます。同名のLSTも読み込みます。
2. `start_static_analysis` に `{"period":"L","result_path":"nakano02_07_mcp.otl","node_ids":[131]}` を渡します。現在開いているINPを一度解析します。結果保存先はサーバーの `-root` からの相対パスで、既存ファイルは上書きしません。節点の数値照会が不要なら `node_ids` は省略できます。
3. `analysis_status` に返された `job_id` を渡して `completed` を確認します。必要に応じて `analysis_results` で指定した節点の変位・反力を取得します。
4. `gui_follow_analysis` に `{"connection_id":"接続ID","job_id":1}` を渡します。IUPにLの変形図とZ方向の変位を表示します。

`displacement_axes` で表示する成分を選べます。たとえば `["X","Z"]` を渡すと両方を表示し、`[]` を渡すと変形図だけを表示します。保存済みの結果は、MCPの再起動後も `open_model` で同じモデルを開き、`gui_show_result` に `{"connection_id":"接続ID","path":"nakano02_07_mcp.otl"}` を渡して再表示できます。

## IUPで材料非線形解析の途中経過を表示する

新しいビルドの `st.exe` と `stmcp.exe` を同じPCで使います。`start_analysis_comparison` の `analysis:"arclm101"` に `preview_dir` を指定すると、編集後モデルの収束済み変位・材端応力・反力をIUPで表示できます。`preview_dir` はMCPの `-root` からの相対パスで、実行ごとに新しいフォルダを指定します。

```json
{"proposal_id":1,"analysis":"arclm101","period":"X","preload_l":true,"node_ids":[131],"steps":80,"increment":0.0125,"preview_dir":"live-nakano-01","preview_every":10}
```

IUPのコマンド欄で、ツールが返す `preview_dir` の絶対パスを開きます。モデルに未保存の変更がある場合は、先に保存してください。

```text
:mcplive "C:/models/live-nakano-01"
```

表示を有効にすると編集後モデルから計算し、その最初の収束ステップを表示します。変更前との比較は続けて計算します。以後は `preview_every`（省略時は10）の間隔で更新し、最後の収束ステップも表示します。高速な計算では画面更新を約200 ms間隔に抑え、表示が追いつかない場合は最新の収束済みステップを表示します。表示周期は `MCP` です。変形と、応力表示が未設定の場合は柱・梁の曲げとブレースの軸力を表示します。たとえば `'stress girder MCP My` で梁の別軸の曲げを表示できます。

`:mcplive off` は画面更新のみを終了します。解析を中止するにはCodexから `cancel_analysis` を呼びます。解析完了・中止後も最後に公開した変形・応力を確認できます。`preview_dir` を省略した解析は従来どおり実行します。

## INPとLSTを一緒に編集・保存する

`open_model` は同名の .lst があれば読み込みます。別名の断面リストを使う場合は `{"path":"model.inp","lst_path":"sections.lst"}` を指定します。`query_lst` で `{"sections":[101]}` のように断面定義を照会できます。`preview_lst_edits` は断面番号と新しい CODE ブロック全体を受け取り、他のブロックを保持します。定義を削除するには `{"section":101,"delete":true}` を指定します。LSTだけを別名に出力するには `save_lst_as` を使います。

RC柱の寸法・配筋を一緒に指定する例です。材料番号を指定すると、既存のINP材料を使います。材料にFCが指定されておらず、既存のLSTにもRC柱定義がない場合は concrete_grade を指定します。

```json
{"section":101,"width_mm":300,"depth_mm":300,"concrete_grade":"FC24","bars":"8-D19","steel":"SD345","cover_mm":40,"hoop":"2x2-D10@100","hoop_steel":"SD295"}
```

これを `preview_rc_column` に渡します。INPの断面諸量とLSTの主筋・帯筋を同じ提案で更新します。既存の降伏耐力値は保持します。続けて別の編集をする場合は、その proposal_id を `preview_gui_edits` または `preview_lst_edits` に渡します。

- `save_model_as` は、LSTを含む提案を同じ別名の .inp・.lst に保存します。
- `overwrite_model` は `{"proposal_id":1}` を受け取り、開いていたファイルを更新します。読み込み後に外部変更された場合は停止します。更新した既存ファイルごとに元の内容を .bak-… に残し、そのパスを返します。

## MCPで時刻歴応答解析を実行する

`start_time_history` を呼び出します。wave はサーバールート相対のCSVで、列順は時刻[s]、X・Y加速度[cm/s²]です。長期荷重Lを先行載荷し、振動モードからRayleigh減衰用の周期を選びます。proposal_id があれば提案モデル、なければ現在モデルで解析します。

```json
{"wave":"wave.csv","output_dir":"run-th-01","damping_ratio":0.02,"node_ids":[131],"preview_every":10}
```

`analysis_status`、`cancel_analysis`、`analysis_results` を既存解析と同じように使います。指定した新規フォルダに .steps.csv・.nodes.csv を出力し、階定義があるモデルには .floors.csv・.stories.csv・.peaks.csv も出力します。preview_every を指定したときは、`:mcplive` または下記の `gui_follow_analysis` で変形・応力を表示できます。

## 起動中のIUPをMCPから操作する

WindowsのSTで `:mcpconnect` を実行し、履歴に表示された接続IDをCodexに伝えます。同じPCのMCPから次を実行できます。

- `gui_status` に connection_id を渡して、開いているモデルと選択を確認します。
- `gui_select` に connection_id と節点・部材番号を渡して、画面上の選択を変更します。
- `gui_apply_proposal` に connection_id と確認済みの proposal_id を渡すと、GUIのモデルをその場で更新します。`gui_get_model` で取り込んだ未保存モデルにも適用できます。取り込み後にGUI側でモデルが変わった場合はエラーを返すので、再取得して提案を作り直してください。適用後はMCPの基準モデルも更新されます。ディスクへの保存はGUIの保存操作で行えます。
- `gui_follow_analysis` に connection_id と job_id を渡すと、そのジョブの途中表示を開始します。GUIに未保存の変更がある場合は、先に作業ファイルを保存して開き直してください。
- `gui_get_model` に connection_id を渡すと、起動中のGUIの未保存モデル・LST定義・節点と部材の選択をMCPの編集基準として取り込みます。先に `open_model` を呼ぶ必要はありません。モデルのパスはMCPの `-root` 内に置いてください。新しい取得は保留中の提案を破棄します。
- 取り込み後は `query_nodes`、`query_elements`、`query_lst` で照会し、`preview_edits` または `preview_gui_edits` で編集を作成して `gui_apply_proposal` で画面に反映します。`preview_gui_edits` で節点・部材番号を省略すると、取得したGUIの選択を使用します。選択をGUIで変更した場合は `gui_get_model` か `gui_status` で更新します。
- `gui_undo` と `gui_redo` に connection_id を渡すと、GUIの既存Undo履歴を使って一操作を戻す・やり直すことができます。GUI操作とMCP操作が同じ履歴に入り、MCP側の基準と選択も更新します。
- `gui_display` は荷重ケース、変形、倍率、変位値、応力成分、視点を即時変更します。省略した項目は保持し、`stress:[]` と `displacement_axes:[]` は各表示を解除します。編集のUndo時も現在の表示設定を維持します。例は下記のとおりです。

```json
{"connection_id":"接続ID","display":{"period":"L","deformation":true,"deformation_scale":100,"displacement_axes":["Z"],"stress":[{"type":"COLUMN","component":"MY"},{"type":"GIRDER","component":"MY"}],"angles":[20,30],"focus":[0,0,3]}}
```

`gui_get_model` と状態応答は `revision` を返します。取り込み後の編集適用・Undo・表示変更ではモデルの一致を照合します。必要に応じて `gui_display`、`gui_undo`、`gui_redo` の `revision` に期待する値を明示できます。解析実行中のモデル取得・編集・Undoは、解析が完了してから行ってください。新しい機能を使うには `st.exe` と `stmcp.exe` の両方を再ビルドして再起動します。

`gui_get_model` の後は、節点荷重・拘束・部材断面の変更を `preview_edits` に最大100件まとめると、`gui_apply_proposal` が差分を一括で反映します。GUIの手編集でモデルが変わったときは適用を拒否し、成功した一括編集は1回のUndoで戻せます。表示や選択だけを照会する場合は `gui_status` または `gui_select` に `"include_revision": false` を渡すと、モデル全体のハッシュ計算を省きます。

- `:mcpconnect off` で接続を終了します。起動中の接続IDは、同じPCで操作を許可するCodexにだけ渡してください。

## 図化ファイルを作る

DXFの図枠付き図面と断面リストは、以下の専用ツールでも作成できます。

- `query_dxf_sections`: 現在の断面形状・材料・配筋・使用階・不足情報を照会します。寸法と座標はmmです。
- `preview_dxf_drawings`: 図面を作成して、ページ構成・縮尺・配置・警告と`draft_id`を返します。この段階では出力ファイルを保存しません。
- `save_dxf_drawings`: 確認したドラフトをルート内の新規`.dxf`に保存します。既存ファイルは上書きしません。

未保存のGUIモデルを使う場合は、先に`gui_get_model`を実行します。モデルやLSTを変更した後は再プレビューが必要です。編集中の提案を図面化する場合は、照会・プレビューに現在の`proposal_id`を指定できます。

```json
{"options":{"title":"構造図","paper":"A3","views":[{"kind":"plan","floor":3},{"kind":"elevation","grid":"Y2"},{"kind":"elevation","grid":"X1"}],"sections":true,"section_columns":3,"section_rows":2}}
```

上記を`preview_dxf_drawings`に渡し、返されたIDを指定して保存します。

```json
{"draft_id":1,"path":"nakano_sheets.dxf"}
```

用紙は`A3`・`A4`（横）、`A3P`・`A4P`（縦）です。`scale`省略/0は図ごとの自動縮尺、正値はモデル図の固定縮尺です。`columns`/`rows`で伏図・軸組図、`section_columns`/`section_rows`で断面カードの配置を指定します。`section_ids`で断面を絞れます。`font`はCADにあるフォント名（既定`msgothic.ttc`）を指定します。

通り・階レベルは現在のモデル内の定義を優先し、なければ同名`.kjn`をルート内で読みます。明示する場合は`kjn_path`を指定します。サーバールート外へはアクセスしません。RC/S柱・梁・鋼製ブレースの断面図に対応し、壁床など未対応の断面図や定義不足は警告・プレースホルダーになります。保存後の再読み込み、ページ・ビューポート・寸法参照の検証を回帰テストに含めています。実モデルの出力は、別途ezdxfによる監査でも確認しています。

GUIの`:drawings sheets.dxf -plans=3 -grids=Y2,X1 -sections`、`:dxfsections sections.dxf`も同じ作図処理を使用します。

`list_fig2` でルート内の既存 .fig2 を探し、`read_fig2` で日本語のタイトルや作図指示を確認できます。モデルを開く前でも呼び出せます。パスは MCP の `-root` からの相対パスです。

`preview_fig2` で既存の図化ファイルをひな形にし、特定の文字列を置換する例です。

```json
{"template":"nakano02_07_model.fig2","replace":[{"find":"解析モデル全体図","text":"解析モデル全体図（確認用）"}]}
```

新規作成では、用紙・表題と作図コマンドをページごとに指定できます。

```json
{"common":["inputfile","viewpoint"],"pages":[
  {"paper":"a4tate","title":"解析モデル全体図","commands":["focus center","angle 15 240","dists 1000 6000","elem column girder brace wall slab"]},
  {"paper":"a4yoko","title":"2階 梁の断面番号","commands":["axonometric","angle 60 0","height 1 2","elem girder","sectcode"]}
]}
```

複数図の組版や繰り返しを使う場合は、Fig2の命令を `content` に直接指定するか、既存ひな形の `for/endfor`、`multiple`、`figure_h` などを流用できます。指定した共通設定と各ページは、保存前に内容を確認できます。`figure_directives` は記述上の図化指示数で、繰り返し展開や組版後のPDFページ数とは異なります。

`save_fig2_as` に `{"path":"nakano02_07_model_mcp.fig2","draft_id":1}` のように渡して、Windowsの日本語版 Fig2 に対応する CP932 形式で別名保存します。PDF出力は従来どおり、図化ファイルと入力モデルを指定して `fig2.exe nakano02_07.inp nakano02_07_model_mcp.fig2` を実行します。応力図などで参照する解析結果や通り芯ファイルは、従来の図化時と同じ場所に用意します。


## 編集中のモデルに材料・断面・節点・線材をまとめて追加する

`run_model_workflow.construct` はIUPのモデルに直接追加します。たとえば次の1回の呼び出しで、SN400のH形鋼を使ったXYグリッドを作成し、平面表示と別名保存まで実行できます。

```json
{
  "connection_id": "初回に渡す接続ID",
  "construct": {
    "properties": [{"id": 101, "material": "SN400"}],
    "sections": [{"id": 501, "property": 101, "name": "H-200x200x8x12", "shape": "HKYOU", "dimensions_mm": [200, 200, 8, 12]}],
    "grid": {"cells": [10, 10, 0], "pitch_m": [1, 1, 0], "section": 501, "type": "GIRDER"}
  },
  "commands": ["'angle 90 -90", "'fit"],
  "save_copy": "xy_grid_SN400.inp",
  "wait_ms": 5000
}
```

任意形状は `nodes: [{"id": 11, "xyz_m": [0,0,0]}]` と `elements: [{"id": 1001, "nodes": [11,12], "section": 501, "type": "BRACE"}]` の形式で指定します。参照する節点・断面・材料は今回追加するものと既存のものを混在させられます。格子の原点は `origin_m`、最初の番号は `first_node` と `first_element` で指定できます。省略時は原点がXYZとも0、番号はモデルの次の空き番号からです。規則格子の節点や部材を1件ずつチャットで列挙する必要はありません。

新規断面の形状はHKYOU・HWEAK・RPIPE（高さ、幅、ウェブ厚、フランジ厚）、CPIPE（外径、厚さ）、PLATE（高さ、幅）をmmで指定します。新規PROPはSTの登録材料名を使い、重量・剛性倍率は1です。各節点の座標と部材長はmです。重複番号、重なる節点・線材、無効な参照は追加前に検出されます。1回分の追加はIUPでまとめてUndoできます。

応答の `construction` に追加件数・全体件数・種類別と方向別の部材数・最短／最長部材長・参照断面の形状寸法とPROP材質が入ります。`save_copy` は任意で、サーバールート内の新しい相対パスに現在の入力と対応するLSTを保存します。GUIでは編集中のモデルを引き続き操作できます。後続の保存だけが失敗した場合、作成済みのモデルには `save_copy` 単独で再試行できます。

`:mcpconnect` は設定先HTTPサーバーの `tools/list` を確認します。古いサーバーが動いている場合はIUPの履歴に表示されます。新版のHTTPサーバーを起動してMCPクライアントを再接続してください。起動中の旧IUPでは、パスのあるモデルに対してスナップショットとリビジョン照合で一括作成できます。無題モデルの直接作成は新版のIUPで利用できます。

## チャットから編集・解析・表示をまとめて実行する

通常のモデル操作ではCodexがSTのMCPツールを直接呼び出します。別の編集操作を探す場合は `list_model_editors` に `{"search":"plate"}` のような検索語を渡せます。対話コマンドは `editor_commands`、ex・Fig2は `commands`、解析には `static_analysis` や専用のジョブツールを使います。サーバーの初期化応答にもこの選び方を記載し、未対応の操作は不足する機能を報告するよう案内しています。

既存のXYグリッドに床を張る場合は、次の引数を1回の `run_model_workflow` に渡します。面の四隅は対象平面内の線材の接続から取得します。節点番号の列挙は不要です。

```json
{
  "connection_id": "STの接続ID",
  "construct": {
    "plate_sections": [
      {"id": 711, "dead_load_kgf_m2": 100, "lload_tf_m2": [0.065, 0.045, 0.035]}
    ],
    "panel_grid": {"plane": "XY", "coordinate_m": 0, "section": 711, "type": "SLAB", "expected_cells": 100}
  }
}
```

`dead_load_kgf_m2` は断面の固定荷重の合計です。材質や厚さを指定せずに新しい断面を定義すると、比重1・弾性係数0の荷重用PROPと等価厚さを使って保存します。上の例は固定荷重0.100 tf/m²と独立した3つのLLOADを持ちます。既存の板断面ではその構造層を保持し、荷重用の層で差額を調整します。既存層の自重を下回る指定はエラーになります。固定荷重とLLOADは `query_sections` でも個別に照会できます。

`panel_grid` は指定平面内の軸方向の線材4辺で閉じた矩形を対象とし、XY・YZ・ZX平面で使えます。`expected_cells` を指定すると件数の不一致で編集全体を取り消します。個別の三角形・凸四角形には、たとえば `"plates":[{"nodes":[11,12,14],"type":"WALL","section":801}]` のように外周順の節点を指定します。断面の変更と面の追加はGUIの1回のUndoで取り消せます。適用した面積は `construction.added_plate_area_m2`、固定・積載荷重は `construction.plate_loads` に断面番号ごとに入ります。

CodexからはSTのMCPツール `run_model_workflow` を直接呼び出します。Fig2コマンドはJSON文字列として渡せます。STが新版のツールを公開していても、更新前から開いているチャットが古い一覧を保持している場合があります。その場合はCodex側のMCP接続を再読み込みし、ツール一覧に `run_model_workflow` と `model_workflow_status` があることを確認します。接続を更新したら最初の呼び出しに同じSTウインドウの接続IDを渡します。

`state: "completed"` のときは、依頼した数値を応答から照合して完了できます。作成時にはGUIから受け取ったモデルの全体件数と `construction.total` をMCP内部で照合します。`construction_revision` と `gui.revision` が同じなら、作成時の件数・部材長・断面・材質は最終モデルにも対応します。後続の編集でリビジョンが変わった場合や、依頼した項目が応答に含まれない場合は、該当する節点・部材などを個別に照会します。最終表示は `gui.display`、指定節点の静的解析結果は `static.nodes` で確認できます。

応答を見失ったときは、同じMCPセッションで `model_workflow_status` に `{}` を渡すと、直近のワークフローの状態・結果が戻ります。複数の依頼を区別する場合は `workflow_id` を指定します。再接続すると以前のセッションの結果は取得できないため、GUI上で依頼に関係する箇所を確認します。停止操作には対象の `workflow_id` が必要です。

IUPで `:mcpconnect` を実行し、表示された接続IDを初回の `run_model_workflow` または `gui_status` に渡します。同じMCPセッションでは以後のGUI操作でIDを省略できます。HTTPでは同じ `Mcp-Session-Id` を使います。別のウインドウに切り替えるときはその接続IDを明示します。

たとえば「節点131の荷重ベクトルを変更し、長期の変形図とZ変位を表示して」と指示した場合、以下を1回の `run_model_workflow` に渡せます。静的解析の出力先にはサーバールートからの相対パスで新しいファイル名を指定します。

```json
{
  "connection_id": "初回に渡す接続ID",
  "edits": [{"node": 131, "load": [1, 2, -2, 0, 0, 0]}],
  "static_analysis": {"period": "L", "result_path": "edited_L.otl", "node_ids": [131]},
  "display": {"deformation_scale": 75, "displacement_axes": ["Z"]}
}
```

実行順は編集、ネイティブコマンド、静的解析と結果表示、表示設定です。必要な項目だけを渡せます。

- 編集には `edits`、現在のGUI選択や対話入力を使う `editor_commands`（`preview_gui_edits.commands` と同形式）、または確認済みの `proposal_id` を指定します。適用時にGUIのモデルと照合します。結果には変更件数と先頭10件の差分が入ります。
- `commands` はex・Fig2の配列です。IUPで必要な準備ができていれば、増分解析と図化を同じ配列に並べられます。解析中はIUPの通常のステップ表示が動き、解析完了時に後続コマンドが進みます。
- `static_analysis` は未保存のモデルから解析し、保存した結果と表示対象のモデルを照合して変形図を表示します。指定節点の変位と反力は応答の `static.nodes` に入ります。
- `wait_ms` は最初の応答までの完了待ちで、既定1500、範囲0～5000です。時間内に終われば最終結果を返します。長い処理は `workflow_id` を返して自動で進みます。状態確認は `model_workflow_status`、停止は `cancel_model_workflow` に同じIDを渡します。どちらも `wait_ms` を指定できます。セッション内では直近のワークフローを参照できます。
- 停止時は後続の処理を取り消し、実行中の静的解析・Arclm101・時刻歴解析に停止要求を送ります。停止機能を持たないネイティブコマンドやモーダルダイアログの実行中は、そのコマンドが戻るまで待ちます。失敗・停止時も完了済みの編集や結果ファイルは残ります。`stages`、`command_job.completed_commands`、`error` で到達点を確認できます。

`mcp_received_at` と `mcp_reply_at` はMCP側の受付と応答準備の時刻です。`stages` には処理ごとの開始・終了時刻、`command_job.timing` にはIUPの各コマンドの開始・終了時刻と描画回数・累計描画時間（ミリ秒）が入ります。チャット入力から最初のツール呼び出しまで、および最終返信までを測る場合は、クライアント側の時刻と突き合わせます。

## 未保存のGUIモデルでex・Fig2コマンドを実行する

`gui_execute_commands` は起動中IUPのモデル・解析結果・LST定義・選択状態に対して、通常のex・Fig2パーサーを実行します。先頭が `:` のexと、先頭が `'` のFig2を同じ配列に指定できます。編集プレビュー用のコマンド一覧による制限はありません。GUIのファイル操作、外部アプリ起動、ダイアログも通常の挙動になります。

断面算定と検定比図の例です。必要なL・X・Yの解析結果をGUIに読み込むか、先に解析を行ってから呼び出します。

```json
{"connection_id":"接続ID","commands":[":srcal","'srcanrate","'srcancolor"]}
```

既定ではツールはすぐに `command_job.job_id` を返します。短い操作は `wait_ms`（最大5000）を渡して完了結果を同時に取得できます。応答が `running` の場合はそのIDを `gui_command_status` に渡します。

```json
{"connection_id":"接続ID","job_id":1}
```

`completed_commands` は完了したコマンド数です。`command`、`progress`、`step`、`steps`、`history`、`error` で進行状況とエラーを確認できます。ログは直近64 KiBを保持します。解析コマンドは終了後に次のコマンドへ進みます。実行中の解析を止めるには、別の `gui_execute_commands` から `commands:[":stop L"]` のように指定します。

未保存モデルの荷重生成から断面算定までを実行する例です。

```json
{"connection_id":"接続ID","commands":[":extractarclm %<.wgt",":analysis -all %<.otl",":srcal","'srcanrate"]}
```

MCPからの `:srcal` はGUI内の現在のLST定義を使用します。ディスクのLSTへ明示的に切り替える場合は `:srcal -reload`、または `:read sections.lst` を指定します。通常のGUI操作から実行した `:srcal` は既存の読み込み規則を使います。

完了時にはMCPの入力モデルと選択も同期します。`rated_elements` は検定比を持つ部材数、`srcan_rate` はGUIの表示フラグです。途中で失敗した場合、それ以前に完了したコマンドの結果はGUIに残ります。モデルの競合を照合するには、`gui_get_model` で取得した `revision` を実行時に指定してください。`sync_error` があればジョブ結果を確認し、モデルのパスをサーバールート内にしてから再取得してください。過去のジョブを再照会しても、同期済みの基準モデルは巻き戻しません。

新版の `st.exe` と `stmcp.exe` を使用し、MCP接続も再起動して新しいツール一覧を読み込みます。


## Googleドライブなどの仮想ドライブとCodexのAppContainer

WindowsでCodexからSTDIOサーバーを起動すると、AppContainer内のプロセスになる場合があります。この環境では仮想Gドライブのファイルアクセスが拒否され、名前付きパイプもデスクトップのSTとは異なる名前空間になることがあります。モデルルートを開けなかった場合も、サーバーは初期化とツール一覧を返し、ファイル操作時に具体的なエラーを返します。

Gドライブをそのまま使う場合は、通常のデスクトップユーザーとしてMCPのローカルHTTPサーバーを起動します。モデル・LST・解析結果は指定したGドライブ上で読み書きします。起動例は次のとおりです。

```powershell
stmcp.exe -root "G:/マイドライブ/st" -http 127.0.0.1:17831 -token-file "C:/path/stmcp-http.token"
```

トークンファイルには32文字以上のランダムな値を保存し、Codexの `mcp_servers.st.url` に `http://127.0.0.1:17831/mcp`、`http_headers.Authorization` に `Bearer ` に続けて同じ値を指定します。既存の `command` と `args` はHTTP設定へ置き換えます。設定後にCodexのMCP接続を再起動してください。

コンソールを出さずに起動する場合は、同じソースを `-ldflags '-H=windowsgui'` で `stmcp-http.exe` としてビルドします。引数なしで起動すると、同じフォルダーの `stmcp-http.json` から下記の設定を読みます。

```json
{
  "root": "G:/マイドライブ/st",
  "address": "127.0.0.1:17831",
  "token_file": "C:/path/stmcp-http.token"
}
```

任意の実行ファイル名では `-http-config` に設定ファイルを指定できます。サーバーはループバックIPに限定して待ち受け、Bearer認証とHost・Origin検証を行います。HTTPセッションごとに編集中モデルを保持し、GUIとの接続は既存の `:mcpconnect` のIDを使います。下記の自動起動設定により、STの `:mcpconnect` から起動できます。


## :mcpconnect 時のHTTPサーバー自動起動

STの起動時にはHTTPサーバーを起動せず、`:mcpconnect` 実行時にのみ起動します。
`st.exe` と同じフォルダーに `stmcp-http.exe` と `stmcp-http.json` を置いてください。
別の場所の設定は環境変数 `ST_MCP_HTTP_CONFIG` にJSONの絶対パスを指定します（実行ファイルはそのJSONと同じフォルダー）。
設定ファイルがない従来のSTDIO構成では自動起動しません。

既に認証可能なST MCPサーバーが起動している場合は再利用します。
Windowsの名前付きミューテックスで起動処理を直列化し、複数STからの同時実行でも二重起動を防止します。
別サービスや異なる認証設定でポートが使われている場合はエラーを表示します。
`:mcpconnect off` やSTの終了では共有HTTPサーバーを終了しません。

### 図面設定の保存と再利用

`query_dxf_settings` → `preview_dxf_settings` → `save_dxf_settings` で、
`.drawing.json` の初期生成・変更前後の確認・保存ができます。
断面共通設定に部材ごとの符号・注記・用紙上の文字位置等を上書きできます。
新しい部材指定は断面・節点の記録を自動付与し、既存設定の対象不一致は拒否します。
設定保存は既存ファイルをバックアップし、レビュー後の変更を検出します。
`query_dxf_sections` と `preview_dxf_drawings` は同名設定を自動で読みます。
別名設定はルート内の `settings_path` で指定できます。
詳細は [図面設定仕様](../../docs/dxf-drawing-settings.md) を参照してください。

図面設定の `style` と `sheet` には製図スタイル・右側タイトル欄・改訂履歴・
図番・表示回転を保存できます。`preview_dxf_drawings.options.views[].rotation`
で個別に上書きでき、レポートの `sheets` で作図領域とタイトル欄を確認できます。
詳細は [製図スタイルと図枠](../../docs/dxf-drafting.md) を参照してください。

### 起動中のIUPでコマンド・解析・表示を一度に実行する

`list_model_editors` の `search` でex・Fig2の構文を検索できます。接続済みのセッション、または `connection_id` を指定した検索では、IUPが読み込んだPGPエイリアスとGUIのコマンド名も対象になります。現在のGUIモデルを解析して表示するときは `run_model_workflow.commands` にGUIで入力する順にコマンドを書きます。変更前後のモデルを両方計算する場合は `start_analysis_comparison` を使います。

次の例は名前の付いたモデルで長期荷重に対する座屈一次モードを計算します。初回は `connection_id` も指定してください。`%<_buckling.otp` と `%<_buckling.wgt` はモデルと同じ場所に作成・更新されます。

```json
{
  "request_id": "buckling-L-01",
  "commands": [
    ":extractarclm %<_buckling.wgt",
    ":bclng001 -period=L -mode=1 -right=100 %<_buckling.otp",
    ":read buckling %<_buckling.otp",
    "'deformation B1"
  ],
  "readback": {"period": "B1", "require_deformation": true, "files": ["%<_buckling.otp"]}
}
```

- `command_job.timing[].analysis.eigenvalues` にソルバの倍精度値、`gui.readback.file_eigenvalue` にファイルの印字精度での値が入ります。`readback.node_ids` で節点ごとの6成分も取得できます。同じジョブで固有値を求めた場合、モードの読み込み世代・ファイル・固有値・非ゼロ変位を照合します。表示と選択は `gui.display`、`gui.node_ids`、`gui.element_ids` に記録します。
- `running` の間は `model_workflow_status` に同じ `workflow_id` を指定します。`include_history: true` で末尾64 KiBまでのログを取得できます。完了後の照会は保存済みの結果を返します。同じMCPセッション内の直近32件が対象です。`request_id` を再送すると同じジョブを返し、内容の異なる再利用や保持期限を過ぎた要求はエラーになります。新しい計算には別のIDを指定します。
- `waiting_for_input` では `command_job.prompt` を読み、`model_workflow_input` に `{"workflow_id": 2, "input": {"prompt_id": 1, "kind": "node", "id": 131}}` のように返します。ファイルは `kind: "file"` と `text` にパス、座標は `kind: "point"` と `point: [x,y,z]` を渡します。確認ダイアログは `choice` の `text` にボタン番号を渡します。利用できる種類は各プロンプトの `kinds` で確認できます。中止は `cancel_model_workflow` です。色やフォントなどネイティブの専用ダイアログでは、待機中の種類が空配列になります。IUP画面でそのダイアログを閉じると続行します。

新しいex・Fig2の分岐を追加したら `go generate ./stlib` で検索一覧を更新してください。`go test ./stlib/internal/commandcatalog` で実装との差分を検出します。
