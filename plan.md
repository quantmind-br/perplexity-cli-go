╭────────────────────────────────────────────────────────────────────────────╮
│                                                                            │
│    Refactoring/Design Plan: Modificação do Subcomando  cookies import      │
│                                                                            │
│   ## 1. Executive Summary & Goals                                          │
│                                                                            │
│   O objetivo principal deste plano é refatorar o comando do CLI para a     │
│   importação de cookies, mudando sua estrutura de subcomando aninhado      │
│   para uma notação de comando e subcomando separados.                      │
│                                                                            │
│   * Comando Atual (Obsoleto):  perplexity cookies import                   │
│   path_to_cookies.json                                                     │
│   * Comando Proposto (Novo):  perplexity import-cookies                    │
│   path_to_cookies.json                                                     │
│                                                                            │
│   ### Key Goals:                                                           │
│                                                                            │
│   1. Mover a lógica de importação de cookies do subcomando  cookies        │
│   import  para um novo comando de nível superior chamado  import-          │
│   cookies .                                                                │
│   2. Preservar a funcionalidade de importação existente, incluindo o       │
│   tratamento de formatos JSON e Netscape e a validação de arquivos.        │
│   3. Remover o subcomando  cookiesImportCmd  do comando pai                │
│   cookiesCmd  após a migração.                                             │
│   4. Garantir que todos os testes unitários relacionados à importação      │
│   de cookies ( cookies_test.go ) sejam atualizados ou reconfigurados       │
│   para validar o novo comando.                                             │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 2. Current Situation Analysis                                         │
│                                                                            │
│   O código para gerenciamento de cookies está centralizado em              │
│   cmd/perplexity/cookies.go .                                              │
│                                                                            │
│   * Comando Principal de Cookies:  var cookiesCmd  é o comando pai         │
│   para todas as operações de cookies ( status ,  clear ,  path ,           │
│   import ).                                                                │
│   * Lógica de Importação:  var cookiesImportCmd  contém toda a lógica      │
│   de importação ( Args: cobra.ExactArgs(1) , leitura de arquivo,           │
│   carregamento via  auth.LoadCookiesFromFile  ou  auth.                    │
│   LoadCookiesFromNetscape , salvamento e mensagens de status).             │
│   * Registro:  cookiesImportCmd  é registrado em  cookiesCmd  via          │
│   cookiesCmd.AddCommand(cookiesImportCmd)  na função  init()  de           │
│   cmd/perplexity/cookies.go .                                              │
│   * Comando Raiz ( rootCmd ):  cookiesCmd  é adicionado ao  rootCmd        │
│   em  cmd/perplexity/root.go  na função  initConfig() .                    │
│   * Testes:  TestCookiesImportCmd_...  em  cmd/perplexity/cookies_test.    │
│   go  dependem da variável  cookiesImportCmd  e sua estrutura.             │
│                                                                            │
│   A refatoração exigirá a extração de  cookiesImportCmd  e sua             │
│   renomeação/reconfiguração para ser um comando de nível superior.         │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 3. Proposed Solution / Refactoring Strategy                           │
│                                                                            │
│   A estratégia consiste em criar um novo comando de nível superior         │
│   importCookiesCmd  que será adicionado diretamente ao  rootCmd , e        │
│   então remover o subcomando antigo.                                       │
│                                                                            │
│   ### 3.1. High-Level Design / Architectural Overview                      │
│                                                                            │
│   O novo comando  import-cookies  será um sibling de  config ,             │
│   history                                                                  │
│   , e  cookies .                                                           │
│                                                                            │
│                                                                            │
│     ----------                                                             │
│     graph TD                                                               │
│         A[rootCmd: perplexity] --> B(config);                              │
│         A --> C(history);                                                  │
│         A --> D(cookies);                                                  │
│         A --> E(version);                                                  │
│         A --> F(import-cookies);                                           │
│         D --> D1(status);                                                  │
│         D --> D2(clear);                                                   │
│         D --> D3(path);                                                    │
│         F --> F1[Implementação de import];                                 │
│     ----------                                                             │
│                                                                            │
│   ### 3.2. Key Components / Modules                                        │
│                                                                            │
│    Componente    | Localização    | Modificação P… | Responsabilid…        │
│   ---------------+----------------+----------------+----------------       │
│    cookies.go    |  cmd/perplexit | Renomear       | Mover a               │
│                  | y/cookies.go   |  cookiesImport | definição do          │
│                  |                | Cmd  para      | comando de            │
│                  |                |  importCookies | importação            │
│                  |                | Cmd , remover  | para a raiz.          │
│                  |                | a lógica do    |                       │
│                  |                |  init()        |                       │
│                  |                | (AddCommand).  |                       │
│    root.go       |  cmd/perplexit | Adicionar o    | Integrar o            │
│                  | y/root.go      | novo           | novo comando          │
│                  |                |  importCookies | de nível              │
│                  |                | Cmd  ao        | superior.             │
│                  |                |  rootCmd .     |                       │
│    cookies.go    |  cmd/perplexit | REMOVER        | Limpar o              │
│                  | y/cookies.go   |  cookiesImport | comando               │
│                  |                | Cmd  do        | obsoleto.             │
│                  |                |  func init() . |                       │
│    cookies_test. |  cmd/perplexit | Atualizar os   | Garantir a            │
│    go            | y/cookies_test | testes         | cobertura de          │
│                  | .go            |  TestCookiesIm | teste                 │
│                  |                | portCmd_...    | contínua.             │
│                  |                | para usar o    |                       │
│                  |                | novo nome da   |                       │
│                  |                | variável       |                       │
│                  |                | ( importCookie |                       │
│                  |                | sCmd ) e a     |                       │
│                  |                | nova estrutura |                       │
│                  |                | de chamada.    |                       │
│                                                                            │
│   ### 3.3. Detailed Action Plan / Phases                                   │
│                                                                            │
│   #### Phase 1: Migração e Renomeação do Comando (M)                       │
│                                                                            │
│   Objective(s): Criar o novo comando de nível superior  import-cookies     │
│   com a lógica existente.                                                  │
│   Priority: High                                                           │
│                                                                            │
│    Task              | Rationale/Goal     | E… | Deliverable/Crite…        │
│   -------------------+--------------------+----+--------------------       │
│    1.1: Criar novo   | Renomear o comando | S  | Em                        │
│    comando de        | aninhado para o    |    |  cmd/perplexity/co        │
│    importação        | novo formato de    |    | okies.go ,                │
│                      | nível superior.    |    | renomear                  │
│                      |                    |    |  cookiesImportCmd         │
│                      |                    |    | para                      │
│                      |                    |    |  importCookiesCmd         │
│                      |                    |    | e alterar seu             │
│                      |                    |    | campo  Use:  para         │
│                      |                    |    |  import-                  │
│    1.2: Mover a      | A lógica de  RunE  | S  | A função  RunE  de        │
│    lógica do comando | deve permanecer a  |    |  importCookiesCmd         │
│                      | mesma.             |    | é idêntica à de           │
│                      |                    |    |  cookiesImportCmd         │
│    1.3: Adicionar ao | Expor o novo       | S  | Em                        │
│     rootCmd          | comando para a     |    |  cmd/perplexity/ro        │
│                      | CLI.               |    | ot.go  no                 │
│                      |                    |    |  func init() ,            │
│                      |                    |    | adicionar                 │
│                      |                    |    |  rootCmd.AddComman        │
│                      |                    |    | d(importCookiesCmd        │
│    1.4: Remover o    | Eliminar o comando | S  | Em                        │
│    registro antigo   | obsoleto do        |    |  cmd/perplexity/co        │
│                      | aninhamento.       |    | okies.go  no              │
│                      |                    |    |  func init() ,            │
│                      |                    |    | remover a linha           │
│                      |                    |    | que adiciona o            │
│                      |                    |    | comando importação        │
│                      |                    |    | ao  cookiesCmd .          │
│                                                                            │
│   #### Phase 2: Atualização dos Testes (S)                                 │
│                                                                            │
│   Objective(s): Garantir que a cobertura de código para a lógica de        │
│   importação seja mantida.                                                 │
│   Priority: High                                                           │
│                                                                            │
│    Task               | Rationale/Goal      | | Deliverable/Criter…        │
│   --------------------+---------------------+-+---------------------       │
│    2.1: Atualizar     | Modificar todos os  | | Em                         │
│    Testes de          | testes que          | |  cmd/perplexity/coo        │
│    Importação         | referenciam         | | kies_test.go ,             │
│                       |  cookiesImportCmd   | | todas as chamadas          │
│                       | para usar a nova    | |  cookiesImportCmd.R        │
│                       | variável            | | unE(...)  e                │
│                       |  importCookiesCmd . | | referências são            │
│                       |                     | | substituídas por           │
│                       |                     | |  importCookiesCmd.R        │
│                       |                     | | unE(...) .                 │
│    2.2: Atualizar     | Verificar se a      | | Em                         │
│    Teste de Estrutura | estrutura do        | |  cmd/perplexity/coo        │
│                       |  cookiesCmd  não    | | kies_test.go , a           │
│                       | contém mais o       | | função                     │
│                       | subcomando          | |  TestCookiesCmdStru        │
│                       |  import .           | | cture  é atualizada        │
│                       |                     | | para remover               │
│                       |                     | | "import" da lista          │
│                       |                     | | de subcomandos             │
│                       |                     | | esperados.                 │
│    2.3: Executar      | Validar que todas   | |  go test ./cmd/perp        │
│    Testes             | as funcionalidades  | | lexity  é executado        │
│                       | de importação       | | com sucesso, com a         │
│                       | continuam           | | mesma cobertura            │
│                       | funcionando no novo | | para a lógica de           │
│                       | comando.            | | importação.                │
│                       |                     | |                            │
│                                                                            │
│   ### 3.4. API Design / Interface Changes                                  │
│                                                                            │
│    API/Interface | Antes          | Depois         | Impacto               │
│   ---------------+----------------+----------------+----------------       │
│    CLI Command   |  perplexity co |  perplexity im | Breaking              │
│                  | okies import < | port-          | Change para           │
│                  | file>          | cookies <file> | usuários              │
│                  |                |                | existentes da         │
│                  |                |                | função de             │
│                  |                |                | importação.           │
│    Variável      |  cookiesImport |  importCookies | Alteração             │
│    Cobra         | Cmd            | Cmd            | interna em            │
│                  |                |                |  cmd/perplexit        │
│                  |                |                | y/cookies.go          │
│                  |                |                | e                     │
│                  |                |                |  cmd/perplexit        │
│    Comando       | Continha       | Contém apenas  | Simplifica o          │
│     cookies      |  import ,      |  status ,      | comando               │
│                  |  status ,      |  clear ,       |  cookies .            │
│                  |  clear ,       |  path .        |                       │
│                  |  path .        |                |                       │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 4. Key Considerations & Risk Mitigation                               │
│                                                                            │
│   ### 4.1. Technical Risks & Challenges                                    │
│                                                                            │
│   * Risco: Confusão de Variáveis/Escopo: A lógica do comando               │
│   cookiesImportCmd  em  cmd/perplexity/cookies.go  usa variáveis           │
│   globais como  cfg ,  render ,  auth ,  os , etc., que devem              │
│   permanecer acessíveis e corretamente configuradas mesmo após a           │
│   mudança de seu comando pai.                                              │
│     * Mitigação: Como o arquivo  cookies.go  já está no mesmo pacote (     │
│     main ) que  root.go  onde as variáveis globais ( cfg ,  render )       │
│     são inicializadas, a refatoração é de baixo risco, pois o escopo       │
│     do pacote é mantido. A única mudança é qual comando Cobra é            │
│     adicionado ao  rootCmd .                                               │
│   * Risco: Quebra de Backward Compatibility: Esta é uma quebra de          │
│   compatibilidade explícita para o comando de importação de cookies.       │
│     * Mitigação: Documentar a mudança nas notas de lançamento (            │
│     CHANGELOG  e  CLAUDE.md , se aplicável) e no  help  do comando.        │
│                                                                            │
│                                                                            │
│   ### 4.2. Dependencies                                                    │
│                                                                            │
│   * Interna (Task-to-Task): A Task 1.4 depende da conclusão da Task 1.     │
│   1 e 1.3. A Fase 2 depende da conclusão da Fase 1.                        │
│   * Interna (Código): O novo  importCookiesCmd  depende das funções de     │
│   autenticação existentes ( auth.LoadCookiesFromFile ,  auth.              │
│   LoadCookiesFromNetscape , etc.).                                         │
│                                                                            │
│   ### 4.3. Non-Functional Requirements (NFRs) Addressed                    │
│                                                                            │
│   * Usabilidade (UX): Simplifica o comando de importação de cookies ao     │
│   elevá-lo para um comando de nível superior. O padrão  perplexity         │
│   import-cookies  é mais direto do que o aninhamento triplo                │
│   perplexity cookies import .                                              │
│   * Manutenibilidade: O comando  cookiesCmd  fica mais focado no           │
│   gerenciamento do estado atual dos cookies (status, clear, path),         │
│   enquanto a importação (uma operação de configuração) é separada,         │
│   melhorando a separação de preocupações de comandos Cobra.                │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 5. Success Metrics / Validation Criteria                              │
│                                                                            │
│   * O comando  perplexity import-cookies path_to_cookies.json  executa     │
│   com sucesso o processo de importação.                                    │
│   * O comando  perplexity cookies import  não é mais um comando válido     │
│   e retorna um erro de Cobra (comando desconhecido).                       │
│   * Os comandos  perplexity cookies status ,  perplexity cookies clear     │
│   e  perplexity cookies path  continuam funcionando corretamente.          │
│   * Todos os testes unitários ( go test ./cmd/perplexity ) passam após     │
│   a refatoração, mantendo a cobertura da lógica de importação.             │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 6. Assumptions Made                                                   │
│                                                                            │
│   * A lógica de  RunE  do  cookiesImportCmd  não precisa de                │
│   modificações (apenas cópia/renomeação), pois a dependência de            │
│   variáveis globais e funções de  auth  está resolvida via escopo do       │
│   pacote.                                                                  │
│   * A intenção do usuário era realmente promover a importação para um      │
│   comando de nível superior, e não simplesmente renomear o subcomando      │
│   dentro do comando  cookies . O comando  import-cookies  sugere um        │
│   comando de raiz.                                                         │
│                                                                            │
│   ## 7. Open Questions / Areas for Further Investigation                   │
│                                                                            │
│   * Nome Exato do Novo Comando: O formato sugerido é  import-cookies .     │
│   A Cobra suporta hífens. Confirma-se que  import-cookies  é o nome        │
│   final desejado em vez de  importcookies  ou  cookie-import .             │
│   (Assunção:  import-cookies  é o formato final).                          │
│   * Necessidade de Alias: Devo manter um alias  import  dentro do          │
│   cookiesCmd  por um período de transição? (Decisão: Não, a                │
│   refatoração visa a remoção completa da aninhagem).                       │
╰────────────────────────────────────────────────────────────────────────────╯