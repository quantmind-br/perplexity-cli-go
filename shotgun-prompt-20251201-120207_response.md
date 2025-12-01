╭────────────────────────────────────────────────────────────────────────────╮
│                                                                            │
│    Refactoring/Design Plan: Aumento da Cobertura de Testes para 80%        │
│                                                                            │
│   ## 1. Executive Summary & Goals                                          │
│                                                                            │
│   O objetivo principal deste plano é identificar as áreas de código        │
│   mais críticas e com menor cobertura de testes no projeto  perplexity-    │
│   cli-go  e implementar novos testes, ou refatorar os existentes, para     │
│   atingir uma cobertura individual mínima de 80% em cada área crítica.     │
│                                                                            │
│   ### Key Goals & Outcomes                                                 │
│                                                                            │
│   * Atingir Cobertura Alvo: Garantir que as áreas críticas (Módulos de     │
│   CLI e Módulos Core de Lógica de Negócio/API) atinjam pelo menos 80%      │
│   de cobertura de testes.                                                  │
│   * Melhorar a Robustez: Aumentar a confiança no código, especialmente     │
│   nas rotinas de entrada/saída de dados ( input/output ), autenticação     │
│   e parsing de resposta da API.                                            │
│   * Reduzir Débito Técnico: Corrigir testes quebrados ou mal               │
│   formatados (ex:  *.broken ).                                             │
│                                                                            │
│   ## 2. Current Situation Analysis                                         │
│                                                                            │
│   A análise da estrutura de arquivos e dos nomes de arquivos de teste      │
│   (incluindo o  coverage.html  e arquivos  *.broken ) sugere as            │
│   seguintes observações:                                                   │
│                                                                            │
│    Área       | Arquivos … | Arquivos … | Cobertura … | Observações        │
│   ------------+------------+------------+-------------+-------------       │
│    A. Core da |  root.go , |  root_test | Baixa/Média | Rotinas de         │
│    CLI        |  cookies.g | .go ,      | (Ex:        |  runQuery          │
│    ( cmd/perp | o ,        |  cookies_t |  config.go  | (em                │
│    lexity )   |  history.g | est.go ,   | 25.0%,      |  root.go )         │
│               | o ,        |  root_inte |  history.go | e a lógica         │
│               |  config.go | gration_te |   8.1%,     | de                 │
│               |            | st.go.brok |  root.go    | subcomandos        │
│               |            | en ,       | 38.3%)      | ( history.g        │
│               |            |  cmd_integ |             | o ,                │
│               |            | ration_tes |             |  config.go         │
│               |            | t.go.broke |             | ) estão            │
│               |            | n          |             | criticament        │
│               |            |            |             | e sub-             │
│               |            |            |             | testadas.          │
│               |            |            |             | Existem            │
│               |            |            |             | testes de          │
│               |            |            |             | integração         │
│               |            |            |             | quebrados.         │
│    B. Auth    |  cookies.g |  cookies_t | Desconhecid | Componente         │
│    Core       | o          | est.go     | a (Mas o    | crucial            │
│    ( internal |            | (6.6KB)    | arquivo     | para               │
│    /auth )    |            |            | existe)     | autenticaçã        │
│               |            |            |             | o (CSRF            │
│               |            |            |             | token,             │
│               |            |            |             | Netscape/JS        │
│               |            |            |             | ON                 │
│               |            |            |             | loading).          │
│               |            |            |             | Deve ser           │
│               |            |            |             | testado            │
│               |            |            |             | exaustivame        │
│               |            |            |             | nte.               │
│               |            |            |             |                    │
│    C. Config  |  manager.g |  manager_t | Desconhecid | Lógica de          │
│    Core       | o          | est.go     | a (Mas o    |  defaults ,        │
│    ( internal |            | (6.1KB)    | arquivo     |  load ,            │
│    /config )  |            |            | existe)     |  save  e           │
│               |            |            |             |  validation        │
│               |            |            |             |   precisa          │
│               |            |            |             | ser                │
│    D. History |  writer.go |  writer_te | Desconhecid | Lógica de          │
│    Core       |            | st.go      | a (Mas o    |  append ,          │
│    ( internal |            | (6.4KB)    | arquivo     |  read  e           │
│    /history ) |            |            | existe)     |  search            │
│               |            |            |             | (incluindo         │
│               |            |            |             | case-              │
│               |            |            |             | insensitivi        │
│               |            |            |             | ty).               │
│    E.         |  client.go |  client_te | Desconhecid | O parsing          │
│    Client/API |  ,         | st.go ,    | a (Arquivos | de SSE             │
│    ( pkg/clie |  http.go , |  http_test |  *.bak  e   | ( search.go        │
│    nt )       |  search.go | .go.bak ,  |  *.broken   |  ) e a             │
│               |  ,         |  search_te | sugerem     | lógica de          │
│               |  upload.go | st.go ,    | problemas)  |  UploadFile        │
│               |            |  search_te |             |                    │
│               |            | st.go.brok |             | ( upload.go        │
│               |            | en ,       |             |  ) são             │
│               |            |  upload_te |             | complexas e        │
│               |            | st.go.bak  |             | críticas.          │
│    F. UI/TUI  |  renderer. |  renderer_ | Desconhecid | Foco em            │
│    ( internal | go ,       | test.go ,  | a (Mas o    | testar a           │
│    /ui )      |  config_me |  config_me | arquivo     | lógica de          │
│               | nu.go      | nu_test.go | existe)     | rendering e        │
│               |            |            |             | formatting,        │
│               |            |            |             | não a              │
│               |            |            |             | interação          │
│               |            |            |             | TUI (que é         │
│               |            |            |             | mais               │
│               |            |            |             | difícil de         │
│               |            |            |             | cobrir).           │
│                                                                            │
│   ### Áreas Mais Críticas para Cobertura                                   │
│                                                                            │
│   1.  cmd/perplexity/root.go  ( runQuery ): Contém a lógica principal      │
│   (prioridade de input, carregamento de cookies, cliente,                  │
│   streaming/non-streaming, history save).                                  │
│   2.  cmd/perplexity/history.go : Os comandos da CLI para visualização     │
│   de histórico.                                                            │
│   3.  pkg/client/search.go : Funções de  buildSearchPayload  e,            │
│   crucialmente, o  parseSSEChunk  e  parseSSEStream .                      │
│   4. Integração CLI-Core: Cobertura de testes de integração para           │
│   cookies ,  config  e  history  commands (usando os mocks existentes      │
│   nos testes quebrados).                                                   │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 3. Proposed Solution / Refactoring Strategy                           │
│                                                                            │
│   O plano será dividido em três fases lógicas: Preparação                  │
│   (Consertar/Organizar), Cobertura Core (Lógica de Negócio) e              │
│   Cobertura CLI (Comandos e Integração).                                   │
│                                                                            │
│   ### 3.1. High-Level Design / Architectural Overview                      │
│                                                                            │
│   A estratégia de testes focará em Testes Unitários para a lógica          │
│   interna e Testes de Integração de Componente para a CLI, explorando      │
│   os mocks e as rotinas de setup existentes (ex:                           │
│   TestCompleteQueryFlow  em  root_integration_test.go.broken ).            │
│                                                                            │
│   * Testes Unitários: Foco em  pkg/client/search.go  ( parseSSEChunk ,     │
│   buildSearchPayload ),  internal/config/manager.go  (validações) e        │
│   internal/history/writer.go  (CRUD, busca).                               │
│   * Testes de Integração de Componente: Foco em cobrir o  RunE  de         │
│   cada comando Cobra ( cookies.go ,  config.go ,  history.go ),            │
│   isolando-o de chamadas HTTP reais (utilizando  MockClient ).             │
│                                                                            │
│   ### 3.2. Key Components / Modules for Testing Focus                      │
│                                                                            │
│    Módulo         | Componentes de… | Requisito … | Tipo de Teste …        │
│   ----------------+-----------------+-------------+-----------------       │
│     pkg/client/se |  parseSSEStream | > 80%       | Unitário               │
│    arch.go        |  ,              |             | (Simulação de          │
│                   |  parseSSEChunk  |             | raw SSE/JSON)          │
│                   | ,               |             |                        │
│                   |  buildSearchPay |             |                        │
│     cmd/perplexit |  runQuery       | > 80%       | Integração (com        │
│    y/root.go      |                 |             | mocks)                 │
│     internal/hist |  ReadLast ,     | > 80%       | Unitário (File         │
│    ory            |  Search ,       |             | IO com                 │
│                   |  Clear          |             |  t.TempDir() )         │
│     internal/auth |  LoadCookiesFro | > 80%       | Unitário               │
│                   | mNetscape ,     |             | (Simulação de          │
│                   |  HasCSRFToken   |             | formatos)              │
│     cmd/perplexit | Comandos        | > 80%       | Integração de          │
│    y              |  cookies ,      |             | Componente             │
│                   |  config ,       |             | (Execução              │
│                   |  history        |             |  RunE )                │
│                                                                            │
│   ### 3.3. Detailed Action Plan / Phases                                   │
│                                                                            │
│   #### Phase 1: Preparação e Limpeza (Foundation)                          │
│                                                                            │
│    Task               | Rationale/Goal     | … | Deliverable/Crite…        │
│   --------------------+--------------------+---+--------------------       │
│    1.1: Recuperar     | Restaurar a base   | S | Arquivos                  │
│    Testes Quebrados   | de testes de       |   |  cmd_integration_t        │
│                       | integração,        |   | est.go.broken  e          │
│                       | renomeando         |   |  root_integration_        │
│                       | arquivos           |   | test.go.broken            │
│                       |  *.broken  para    |   | renomeados para           │
│                       |  *.go .            |   |  _test.go  e              │
│                       |                    |   | corrigidos para           │
│                       |                    |   | rodar sem erros.          │
│    1.2: Normalizar    | Unificar/Refatorar | S | Um único                  │
│    Mocks              | o  MockClient  em  |   |  MockClient  (ou          │
│                       |  cmd/perplexity/cm |   | Interface)                │
│                       | d_integration_test |   | definido e usado          │
│                       | .go  com           |   | por todos os              │
│                       |  FullMockClient    |   | testes de                 │
│                       | em                 |   | integração.               │
│                       |  cmd/perplexity/ro |   |                           │
│                       | ot_integration_tes |   |                           │
│                       | t.go  (mantendo o  |   |                           │
│    1.3: Adicionar     | Adicionar testes   | M | Cobertura de              │
│    Testes Básicos     | para  init         |   |  cmd/perplexity/co        │
│    Faltantes na CLI   | functions e help   |   | nfig.go ,                 │
│                       | text/estrutura de  |   |  cmd/perplexity/hi        │
│                       | comandos não       |   | story.go ,                │
│                       | cobertos em        |   |  cmd/perplexity/ve        │
│                       |  config.go ,       |   | rsion.go                  │
│                       |  history.go ,      |   | atingindo o alvo.         │
│                       |  main.go ,         |   |                           │
│                       |  version.go        |   |                           │
│                       | (visando 80% de    |   |                           │
│                       | cobertura nesses   |   |                           │
│                       | arquivos).         |   |                           │
│                                                                            │
│   #### Phase 2: Cobertura Core (Client e Parsing)                          │
│                                                                            │
│    Task               | Rationale/Goal      | | Deliverable/Criter…        │
│   --------------------+---------------------+-+---------------------       │
│    2.1: Aumentar      | Cobrir todos os     | | Cobertura de               │
│    Cobertura em       | caminhos e formatos | |  pkg/client/search.        │
│     pkg/client/search | de chunk (DONE,     | | go  > 80%.                 │
│    .go                | delta, text, legacy | |                            │
│    ( parseSSEChunk )  | blocks, new step-   | |                            │
│                       | based format,       | |                            │
│                       | errors).            | |                            │
│    2.2: Aumentar      | Focar em            | | Cobertura de               │
│    Cobertura em       |  LoadCookiesFromNet | |  internal/auth/cook        │
│     internal/auth/coo | scape  e cobrir     | | ies.go  > 80%.             │
│    kies.go            | caminhos de erro e  | |                            │
│                       | edge cases          | |                            │
│                       | (expiração,         | |                            │
│                       | SameSite, domain    | |                            │
│                       | filtering).         | |                            │
│    2.3: Aumentar      | Cobrir rotinas de   | | Cobertura de               │
│    Cobertura em       |  Load  e  validate  | |  internal/config/ma        │
│     internal/config/m | (modelos, modos,    | | nager.go  > 80%.           │
│    anager.go          | formatos de         | |                            │
│                       | linguagem           | |                            │
│                       | inválidos, fontes). | |                            │
│    2.4: Aumentar      | Cobrir rotinas de   | | Cobertura de               │
│    Cobertura em       |  ReadLast ,         | |  internal/history/w        │
│     internal/history/ |  Search  (com case- | | riter.go  > 80%.           │
│    writer.go          | insensitivity),     | |                            │
│                       |  Clear  e error     | |                            │
│                       | handling de IO.     | |                            │
│    2.5: Consertar e   | Restaurar e         | | Cobertura de               │
│    Aumentar Cobertura | expandir            | |  pkg/client/upload.        │
│    em                 |  upload_test.go.bak | | go  > 80% e                │
│     pkg/client/upload |  . Cobrir           | | correção de                │
│    .go                |  detectContentType  | |  upload_test.go.bak        │
│                       | ,  isImageFile , e  | |  .                         │
│                       | a lógica de         | |                            │
│                       |  rewriteImageURL .  | |                            │
│                                                                            │
│   #### Phase 3: Cobertura de Lógica de Negócio e Integração (CLI)          │
│                                                                            │
│    Task               | Rationale/Goal      | | Deliverable/Criter…        │
│   --------------------+---------------------+-+---------------------       │
│    3.1: Cobertura de  | Implementar testes  | | Cobertura de               │
│     cmd/perplexity/ro | de integração       | |  cmd/perplexity/roo        │
│    ot.go              | isolados (com       | | t.go  > 80%.               │
│    ( runQuery )       |  MockClient ) para  | |                            │
│                       | cobrir todos os     | |                            │
│                       | caminhos de         | |                            │
│                       |  runQuery : input   | |                            │
│                       | priority (-f > args | |                            │
│                       | > stdin),           | |                            │
│                       | streaming/non-      | |                            │
│                       | streaming,          | |                            │
│                       | incognito, verbose, | |                            │
│                       | save to file.       | |                            │
│    3.2: Cobertura de  | Cobrir todos os     | | Cobertura de               │
│    Comandos  cookies  | caminhos  RunE  de  | |  cmd/perplexity/coo        │
│                       |  cookiesStatusCmd , | | kies.go  > 80%.            │
│                       |  cookiesClearCmd ,  | |                            │
│                       |  importCookiesCmd   | |                            │
│                       | (file not found,    | |                            │
│                       | load/save errors,   | |                            │
│                       | missing CSRF        | |                            │
│                       | token).             | |                            │
│    3.3: Cobertura de  | Cobrir todos os     | | Cobertura de               │
│    Comandos  history  | caminhos  RunE  de  | |  cmd/perplexity/his        │
│                       |  historyListCmd ,   | | tory.go  > 80%.            │
│                       |  historySearchCmd , | |                            │
│                       |  historyShowCmd     | |                            │
│                       | (index out of       | |                            │
│                       | range, search no    | |                            │
│                       | results, display    | |                            │
│                       | content).           | |                            │
│    3.4: Cobertura de  | Focar em            | | Cobertura de               │
│    TUI/Renderer       |  internal/ui/render | |  internal/ui/render        │
│                       | er.go  para cobrir  | | er.go  > 80%.              │
│                       |  RenderResponse     | |                            │
│                       | com novos formatos  | |                            │
│                       | (Text + WebResults) | |                            │
│                       | e                   | |                            │
│                       |  RenderWebResults   | |                            │
│                       | (filtragem de       | |                            │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 4. Key Considerations & Risk Mitigation                               │
│                                                                            │
│   ### 4.1. Technical Risks & Challenges                                    │
│                                                                            │
│    Risco               | Descrição           | Mitigação                   │
│   ---------------------+---------------------+----------------------       │
│    Parsing de SSE      |  pkg/client/search. | Uso extensivo de            │
│    Complexo            | go  lida com        | testes de tabela em         │
│                        | múltiplos formatos  |  parseSSEChunk  e           │
│                        | de SSE (legacy      |  parseStepBasedRespo        │
│                        | blocks vs. new      | nse  com raw strings        │
│                        | step-based format,  | simulando o output          │
│                        | JSON aninhado).     | do servidor para            │
│                        |                     | garantir a robustez         │
│                        |                     | do parsing.                 │
│    Integração de Mocks | Os testes de        | Refatorar os mocks          │
│                        | integração CLI      | existentes na Fase 1        │
│                        | dependem de mocks   | para garantir que a         │
│                        | robustos do         | interface                   │
│                        |  client.Client . Se |  client.Client  e as        │
│                        | a interface do mock | implementações de           │
│                        | for insuficiente,   | mock suportem todas         │
│                        | os testes falharão  | as chamadas                 │
│                        | em simular o        | necessárias                 │
│                        | comportamento real. | ( Search ,                  │
│                        |                     |  SearchStream ,             │
│    Testes de TUI       | Implementar testes  | Priorizar testes            │
│                        | para a TUI          | apenas para as              │
│                        | ( internal/ui/confi | funções auxiliares          │
│                        | g_menu.go ) é       | ( buildConfigMenuIte        │
│                        | complexo e de baixo | ms ,                        │
│                        | retorno em Go.      |  getModeDescription         │
│                        |                     | ) e focar em cobrir         │
│                        |                     | a lógica de                 │
│                        |                     | rendering                   │
│                        |                     | ( renderer.go ) em          │
│                        |                     | vez da interação TUI        │
│                                                                            │
│   ### 4.2. Dependencies                                                    │
│                                                                            │
│   * Conclusão da Phase 1 (Preparação) é um pré-requisito para as           │
│   Phases 2 e 3.                                                            │
│   * A Cobertura da CLI (Phase 3) depende da correção dos módulos Core      │
│   (Phase 2), especialmente a lógica de history e cookies.                  │
│                                                                            │
│   ### 4.3. Non-Functional Requirements (NFRs) Addressed                    │
│                                                                            │
│   * Maintainability: Testes bem definidos tornam o código mais fácil       │
│   de entender e modificar.                                                 │
│   * Reliability: Cobertura de 80%+ nas áreas críticas garante que a        │
│   maioria dos caminhos de execução (incluindo erros e edge cases)          │
│   sejam validados.                                                         │
│   * Testability: A refatoração implícita para cobrir o código com          │
│   testes forçará uma melhor separação de responsabilidades,                │
│   facilitando futuros testes (e mantendo a DI/interface do  client ).      │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 5. Success Metrics / Validation Criteria                              │
│                                                                            │
│   * Quantitative (Principal): A saída do  go tool cover -func=coverage.    │
│   out  deve mostrar que os seguintes arquivos atingiram cobertura de       │
│   declaração (statement coverage) de 80% ou mais:                          │
│     *  cmd/perplexity/root.go                                              │
│     *  cmd/perplexity/cookies.go                                           │
│     *  cmd/perplexity/history.go                                           │
│     *  internal/auth/cookies.go                                            │
│     *  internal/config/manager.go                                          │
│     *  internal/history/writer.go                                          │
│     *  pkg/client/search.go                                                │
│     *  pkg/client/upload.go                                                │
│     *  internal/ui/renderer.go                                             │
│   * Qualitative: Os testes de integração quebrados ( *.broken ) foram      │
│   corrigidos e reintroduzidos na suíte principal sem falhas.               │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 6. Assumptions Made                                                   │
│                                                                            │
│   * Assume-se que a ferramenta de cobertura de código Go ( go tool         │
│   cover ) está disponível e será usada para medir o sucesso.               │
│   * Assume-se que os testes existentes (não marcados como  *.broken        │
│   ou  *.bak ) são funcionais e podem ser expandidos.                       │
│   * Assume-se que a cobertura de 80% é um objetivo razoável e              │
│   alcançável para a maior parte da lógica de negócio.                      │
│                                                                            │
│   ## 7. Open Questions / Areas for Further Investigation                   │
│                                                                            │
│   * Cobertura do  http.go : O arquivo  pkg/client/http.go  é um            │
│   wrapper para um cliente TLS de terceiros. Qual é a expectativa de        │
│   cobertura? Recomenda-se testar apenas as funções utilitárias (           │
│   buildHeaders ,  normalizeURL ) e as rotinas de cookie/CSRF token que     │
│   manipulam o cliente, não as chamadas de rede externas.                   │
│   * Configuração Global na CLI: Os testes de integração usam e             │
│   modificam variáveis globais ( cfg ,  cfgMgr ,  flag* ). É necessário     │
│   isolar o ambiente de teste em cada teste de integração (ex: usando       │
│   um  setupTestEnv  mais robusto) para evitar efeitos colaterais entre     │
│   testes.                                                                  │
╰────────────────────────────────────────────────────────────────────────────╯