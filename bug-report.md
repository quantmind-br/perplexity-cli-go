╭───────────────────────────────────────────────────────────────────────────────╮
│                                                                               │
│    Bug Analysis Report: Quebras de Linha Incorretas na Resposta               │
│   Renderizada (Word Wrap Issue)                                               │
│                                                                               │
│   ## 1. Executive Summary                                                     │
│                                                                               │
│   O bug relatado manifesta-se como quebras de linha incorretas e              │
│   truncamento de texto (ex: palavras quebradas, caracteres únicos             │
│   deslocados) dentro da caixa estilizada que encapsula a resposta da          │
│   IA.                                                                         │
│                                                                               │
│   O problema principal reside em um erro de cálculo de largura ( width        │
│   mismatch ) entre a biblioteca de renderização Markdown interna (            │
│   glamour ) e a biblioteca de estilização e bordas externa ( lipgloss         │
│   ). A largura total da janela do terminal é passada para o  glamour          │
│   para quebra de palavras, mas não é subtraído o espaço ocupado pelas         │
│   bordas e preenchimentos da caixa de resposta.                               │
│                                                                               │
│   * Causa Mais Provável: A largura utilizada pela função de quebra de         │
│   palavra do Markdown ( glamour.WithWordWrap ) é maior que a área de          │
│   conteúdo disponível dentro da moldura da resposta (criada por               │
│   lipgloss.NewStyle().Border().Padding() ).                                   │
│   * Módulos Chave Envolvidos:  internal/ui/renderer.go                        │
│   (especificamente  NewRendererWithOptions  e  RenderStyledResponse ).        │
│                                                                               │
│   --------                                                                    │
│                                                                               │
│   ## 2. Bug Description and Context (from  User Task )                        │
│                                                                               │
│   * Comportamento Observado: A resposta da IA, renderizada dentro de          │
│   uma moldura ASCII, apresenta quebras de linha anormais, truncando           │
│   palavras ou deixando caracteres isolados nas margens da caixa.              │
│     * Exemplo: O caractere "O" é isolado na linha 17 (de "O que               │
│     existem").                                                                │
│     * Exemplo: A palavra "uma" é truncada na linha 28.                        │
│     * Exemplo: A palavra "descoberta" é truncada na linha 45.                 │
│   * Comportamento Esperado: O texto deve ser formatado em Markdown,           │
│   com quebra de palavra suave e correta, ajustando-se à largura               │
│   disponível dentro das margens e bordas da caixa estilizada.                 │
│   * Steps to Reproduce (STR): Executar qualquer consulta que resulte          │
│   em um parágrafo longo o suficiente para exigir quebra de linha em um        │
│   terminal de largura limitada (ex: 80 colunas).                              │
│    perplexity "salomão existiu? existem comprovações?"                        │
│   * Ambiente: CLI  perplexity  (Go).                                          │
│                                                                               │
│   --------                                                                    │
│                                                                               │
│   ## 3. Code Execution Path Analysis                                          │
│                                                                               │
│   ### 3.1. Entry Point(s) and Initial State                                   │
│                                                                               │
│   A execução começa em  cmd/perplexity/root.go  na função  runQuery ,         │
│   que chama  render.RenderResponse(resp) . Por sua vez, esta função           │
│   chama  render.RenderStyledResponse(resp.Text)  (ou similar,                 │
│   dependendo do formato de resposta da API).                                  │
│                                                                               │
│   ### 3.2. Key Functions/Modules/Components in the Execution Path             │
│                                                                               │
│   1.  internal/ui/renderer.go:NewRendererWithOptions : Responsável por        │
│   configurar o renderer Markdown ( glamour ), incluindo a regra de            │
│   quebra de palavra ( glamour.WithWordWrap ).                                 │
│   2.  internal/ui/renderer.go:ResponseContainerStyle : Define as              │
│   características visuais da caixa de resposta (borda, padding).              │
│   3.  internal/ui/renderer.go:RenderStyledResponse : Recebe o texto,          │
│   envia para renderização Markdown (onde ocorre o word wrap) e, em            │
│   seguida, aplica a estilização da caixa ao resultado.                        │
│                                                                               │
│   ### 3.3. Execution Flow Tracing                                             │
│                                                                               │
│    Passo | Módulo/Função    | Ação              | Resultado/Presun…           │
│   -------+------------------+-------------------+-------------------          │
│      1   |  internal/ui/ren | Define o estilo   |  Padding(1, 2)  +           │
│          | derer.go:Respons | da caixa.         |  Border()                   │
│          | eContainerStyle  |                   | implica 6 colunas           │
│          |                  |                   | de overhead                 │
│          |                  |                   | horizontal (2 de            │
│          |                  |                   | borda + 4 de                │
│          |                  |                   | padding).                   │
│      2   |  internal/ui/ren | Inicializa o      |  glamour.WithWord           │
│          | derer.go:NewRend |  glamour.TermRend | Wrap(width)  é              │
│          | ererWithOptions  | erer .            | chamado usando a            │
│          |                  |                   | largura total do            │
│          |                  |                   | terminal                    │
│          |                  |                   | ( r.width , ex:             │
│          |                  |                   | 80). Assunção               │
│          |                  |                   | Crítica: O valor            │
│          |                  |                   |  width  (ex: 80)            │
│          |                  |                   | deveria ser                 │
│          |                  |                   | subtraído pelo              │
│          |                  |                   | overhead da                 │
│          |                  |                   | moldura (6                  │
│          |                  |                   | colunas).                   │
│      3   |  r.mdRender.Rend | O Markdown é      | O conteúdo é                │
│          | er(normalizedCon | renderizado e     | quebrado em                 │
│          | tent)            | quebrado.         | linhas de 80                │
│          |                  |                   | colunas                     │
│          |                  |                   | (incorreto). A              │
│          |                  |                   | largura correta             │
│          |                  |                   | deveria ser 74              │
│          |                  |                   | colunas ( 80 -              │
│          |                  |                   |  6 ).                       │
│      4   |  ResponseContain | A Lipgloss aplica | A Lipgloss tenta            │
│          | erStyle.Width(r. | a moldura ao      | encaixar o                  │
│          | width).Render(re | conteúdo          | conteúdo de 80              │
│          | ndered)          | renderizado.      | colunas (do                 │
│          |                  |                   |  glamour ) em um            │
│          |                  |                   | espaço de apenas            │
│          |                  |                   | 74 colunas (80 -            │
│          |                  |                   | 6 colunas de                │
│          |                  |                   | overhead),                  │
│          |                  |                   | forçando quebras            │
│          |                  |                   | ou deslocamento.            │
│      5   | Bug Manifestado  | O primeiro texto  |                             │
│          |                  | de cada linha que |                             │
│          |                  | excede o limite   |                             │
│          |                  | de 74 colunas é   |                             │
│          |                  | empurrado para a  |                             │
│          |                  | margem da caixa,  |                             │
│          |                  | causando o        |                             │
│          |                  | corte/deslocament |                             │
│          |                  | o dos primeiros   |                             │
│          |                  | caracteres da     |                             │
│          |                  | linha (o que      |                             │
│          |                  | acontece com  O , |                             │
│          |                  |  uma ,            |                             │
│          |                  |  descoberta ).    |                             │
│                                                                               │
│   ### 3.4. Data State and Flow Analysis                                       │
│                                                                               │
│   * Variável Crítica:  width  (definida no  Renderer  e usada em              │
│   NewRendererWithOptions ).                                                   │
│   * Estado Incorreto:  mdRender  é configurado para uma largura de 80,        │
│   quando a largura real de conteúdo disponível para ele é de 74. O            │
│   pipeline de renderização falha em sincronizar os limites de quebra          │
│   de palavra interna ( glamour ) com os limites de exibição da moldura        │
│   externa ( lipgloss ).                                                       │
│                                                                               │
│   --------                                                                    │
│                                                                               │
│   ## 4. Potential Root Causes and Hypotheses                                  │
│                                                                               │
│   ### 4.1. Hypothesis 1: Width Mismatch in Glamour Initialization             │
│   (Most Likely Cause)                                                         │
│                                                                               │
│   * Rationale/Evidence: A  ResponseContainerStyle  utiliza  Padding(1,        │
│   2) . O  lipgloss.Border()  implicitamente adiciona 1 coluna de              │
│   largura em cada lado.                                                       │
│     *  Total Overhead Horizontal = 2 * (Border Width) + 2 * (Padding          │
│     Width) = 2*1 + 2*2 = 6 colunas .                                          │
│     * O código falha ao subtrair esse overhead da largura total (             │
│     width ) antes de configurar o  glamour.TermRenderer .                     │
│   * Código Relevante ( internal/ui/renderer.go ):                             │
│                                                                               │
│     ----------                                                                │
│     // Linhas 121-125                                                         │
│     mdRender, err := glamour.NewTermRenderer(                                 │
│         glamour.WithAutoStyle(),                                              │
│         glamour.WithStylePath("dracula"),                                     │
│         glamour.WithWordWrap(width), // <--- O ERRO ESTÁ AQUI: usa            │
│   'width' (ex: 80)                                                            │
│         glamour.WithStylePath(style),                                         │
│     )                                                                         │
│     // ...                                                                    │
│     // Linha 93                                                               │
│     ResponseContainerStyle = lipgloss.NewStyle().                             │
│         Border(lipgloss.RoundedBorder()). // Adiciona 1 coluna de borda       │
│   L/R                                                                         │
│         BorderForeground(WarmColorPrimary).                                   │
│         Padding(1, 2). // Adiciona 2 colunas de padding L/R                   │
│         Margin(1, 0, 0, 0) // <--- O overhead de 6 colunas é definido         │
│   aqui.                                                                       │
│     ----------                                                                │
│                                                                               │
│   * Mecanismo do Bug:  glamour  quebra a linha em 80 colunas. A               │
│   Lipgloss só tem 74 colunas para encaixar este conteúdo (dentro das 6        │
│   colunas de borda/padding/borda), resultando no recorte do início de         │
│   cada linha.                                                                 │
│                                                                               │
│   --------                                                                    │
│                                                                               │
│   ## 6. Recommended Steps for Debugging and Verification                      │
│                                                                               │
│   O bug é determinístico, baseado na lógica de cálculo de layout.             │
│                                                                               │
│   ### A. Correção Direta (Implementar)                                        │
│                                                                               │
│   Modificar a função  internal/ui/renderer.go:NewRendererWithOptions          │
│   para calcular a largura efetiva do conteúdo:                                │
│                                                                               │
│   1. Identificar o Overhead: Determinar o overhead horizontal fixo da         │
│   ResponseContainerStyle .                                                    │
│     * Borda arredondada (1 coluna esq/dir) = 2                                │
│     * Padding horizontal (2 colunas esq/dir) = 4                              │
│     * Total Overhead = 6                                                      │
│   2. Ajustar a Largura: Corrigir a inicialização do  glamour.                 │
│   TermRenderer .                                                              │
│                                                                               │
│                                                                               │
│     ----------                                                                │
│     // internal/ui/renderer.go                                                │
│                                                                               │
│     // ...                                                                    │
│     func NewRendererWithOptions(out io.Writer, width int, useColors           │
│   bool) (*Renderer, error) {                                                  │
│         style := "dark"                                                       │
│         if !useColors {                                                       │
│             style = "notty"                                                   │
│         }                                                                     │
│                                                                               │
│         // Calcula a largura efetiva para o conteúdo interno,                 │
│   subtraindo 6 colunas (2 bordas + 4 padding)                                 │
│         effectiveContentWidth := width - 6                                    │
│         if effectiveContentWidth < 1 {                                        │
│             effectiveContentWidth = 1                                         │
│         }                                                                     │
│                                                                               │
│         mdRender, err := glamour.NewTermRenderer(                             │
│             glamour.WithAutoStyle(),                                          │
│             glamour.WithStylePath("dracula"),                                 │
│             glamour.WithWordWrap(effectiveContentWidth), // FIX: Usar a       │
│   largura ajustada                                                            │
│             glamour.WithStylePath(style),                                     │
│         )                                                                     │
│     // ...                                                                    │
│     ----------                                                                │
│                                                                               │
│   ### B. Teste de Verificação (Recomendado)                                   │
│                                                                               │
│   * Test Scenario:                                                            │
│     1. Defina a largura do terminal para 80 (                                 │
│     NewRendererWithOptions(os.Stdout, 80, true) ).                            │
│     2. Passe uma string longa de Markdown para  RenderStyledResponse .        │
│     3. Verifique se o texto é quebrado em linhas de 74 caracteres e se        │
│     a saída final na tela de 80 colunas não apresenta quebras ou              │
│     truncamentos de palavras na margem.                                       │
│                                                                               │
│                                                                               │
│   --------                                                                    │
│                                                                               │
│   ## 7. Bug Impact Assessment                                                 │
│                                                                               │
│   O impacto do bug é MÉDIO, afetando diretamente a usabilidade e a            │
│   experiência do usuário (UX). O conteúdo da resposta permanece               │
│   correto, mas a formatação visual inadequada prejudica severamente a         │
│   legibilidade, o que é crítico para uma ferramenta de linha de               │
│   comando baseada em texto.                                                   │
│                                                                               │
│   --------                                                                    │
│                                                                               │
│   ## 8. Assumptions Made During Analysis                                      │
│                                                                               │
│   1. A largura total do overhead da  ResponseContainerStyle  é de 6           │
│   colunas (1 coluna de borda  lipgloss.RoundedBorder()  + 2 colunas de        │
│   Padding(1, 2)  em cada lado horizontal).                                    │
│   2. A variável  r.width  (o parâmetro  width  em                             │
│   NewRendererWithOptions ) representa a largura total do terminal.            │
│   3. A  glamour  (o  mdRender ) é responsável pela quebra de linha do         │
│   conteúdo, que deve se ajustar à largura do container (Lipgloss).            │
│   4. O trecho  normalizeMarkdownText(content)  está funcionando               │
│   corretamente, como um pré-processamento de "des-quebra de linha" do         │
│   texto de entrada, e o problema é a re-quebra incorreta subsequente.         │
│   5. A solução reside em sincronizar as larguras de  glamour.                 │
│   WithWordWrap  e  ResponseContainerStyle .                                   │
╰───────────────────────────────────────────────────────────────────────────────╯