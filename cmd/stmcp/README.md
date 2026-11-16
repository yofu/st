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
- `gui_apply_proposal` に connection_id と確認済みの proposal_id を渡すと、GUIのモデルをその場で更新します。開いているモデルが提案の基準と異なるか、未保存の変更があるとエラーを返します。画面に適用した変更をディスクへ保存する場合は、MCPの保存ツールを使います。
- `gui_follow_analysis` に connection_id と job_id を渡すと、そのジョブの途中表示を開始します。GUIに未保存の変更がある場合は、先に作業ファイルを保存して開き直してください。
- `:mcpconnect off` で接続を終了します。起動中の接続IDは、同じPCで操作を許可するCodexにだけ渡してください。

## 図化ファイルを作る

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
