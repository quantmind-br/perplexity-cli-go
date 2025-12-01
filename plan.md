╭────────────────────────────────────────────────────────────────────────────╮
│                                                                            │
│    Refactoring/Design Plan: Implementação de Flags de I/O ( -o ,  -f )     │
│                                                                            │
│   ## 1. Executive Summary & Goals                                          │
│                                                                            │
│   O objetivo deste plano é aprimorar a usabilidade do Perplexity CLI,      │
│   adicionando funcionalidades para salvar a saída da query diretamente     │
│   em um arquivo ( -o / --output ) e ler o conteúdo da query a partir       │
│   de                                                                       │
│   um arquivo ( -f / --file ). A implementação deve ser feita usando o      │
│   framework Cobra existente.                                               │
│                                                                            │
│   ### Key Goals:                                                           │
│                                                                            │
│   1. Adicionar a flag  -o  ou  --output  para salvar a resposta            │
│   completa em um arquivo  .md  ou  .txt .                                  │
│   2. Adicionar a flag  -f  ou  --file  para ler o conteúdo da query a      │
│   partir de um arquivo, priorizando essa leitura sobre a query             │
│   fornecida nos argumentos de linha de comando.                            │
│   3. Garantir a compatibilidade do novo fluxo de input ( -f ) com a        │
│   nova funcionalidade de output ( -o ).                                    │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 2. Current Situation Analysis                                         │
│                                                                            │
│   O núcleo do CLI é o  cmd/perplexity/root.go , onde a função              │
│   runQuery  orquestra a obtenção da query, a configuração das opções       │
│   de busca, a execução da busca e o salvamento da resposta no              │
│   histórico.                                                               │
│                                                                            │
│   * Output Saving: A lógica para salvar o output ( flagOutputFile ,        │
│   $307 ) já existe no final de  runQuery . No entanto, o  User Task        │
│   exige a formalização da flag  -o  e a confirmação de que funciona        │
│   para  .md  e  .txt . A implementação atual usa  os.WriteFile  com o      │
│   flagOutputFile , o que é apropriado.                                     │
│   * Query Input: A query é obtida pela função  getQueryFromInput(args,     │
│   os.Stdin, isTerminal)  no  $153  de  cmd/perplexity/root.go , que        │
│   prioriza os argumentos de linha de comando ( args ) e, como fallback,    │
│   lê do  os.Stdin  se não for um terminal. A nova flag  -f / --file        │
│   requer uma nova fonte de input que deve ser priorizada ou tratada        │
│   antes da lógica de  getQueryFromInput .                                  │
│   * Flags: A flag  flagOutputFile  já está declarada, mas a flag           │
│   flagFile  (para input de arquivo) precisa ser criada e integrada ao      │
│   rootCmd .                                                                │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 3. Proposed Solution / Refactoring Strategy                           │
│                                                                            │
│   A estratégia se concentrará em (1) adicionar a nova flag e (2)           │
│   refatorar a obtenção da query para incorporar a leitura de arquivo       │
│   antes de verificar argumentos ou stdin.                                  │
│                                                                            │
│   ### 3.1. High-Level Design / Architectural Overview                      │
│                                                                            │
│   O novo fluxo de obtenção da query dentro de  runQuery  será:             │
│                                                                            │
│   1. Verificar Flag  -f / --file : Se a flag estiver presente, ler o       │
│   conteúdo do arquivo especificado. Se bem-sucedido, este é o  query       │
│   final.                                                                   │
│   2. Fallback para Argumentos/Stdin: Se a flag  -f  não estiver            │
│   presente ou estiver vazia, usar a lógica existente (                     │
│   getQueryFromInput ) para obter a query de  args  ou  stdin .             │
│   3. Execução da Query: Se uma query válida for encontrada, prosseguir     │
│   com a execução normal.                                                   │
│                                                                            │
│                                                                            │
│     ----------                                                             │
│     graph TD                                                               │
│         A[Start runQuery] --> B{Flag -f ou --file presente?};              │
│         B -- Sim --> C[Ler Conteúdo do Arquivo];                           │
│         C --> D{Leitura OK e Não Vazia?};                                  │
│         D -- Sim --> G[query = Conteúdo do Arquivo];                       │
│         D -- Não --> F[Tratar Erro/Query Vazia (Mostrar Ajuda)];           │
│         B -- Não --> E{Obter Query de args/stdin (getQueryFromInput)};     │
│         E --> H{query Vazia?};                                             │
│         H -- Sim --> F;                                                    │
│         H -- Não --> I[query = args/stdin content];                        │
│         G --> J[Continuar com Execução da Query];                          │
│         I --> J;                                                           │
│         F --> K[Fim com Ajuda/Erro];                                       │
│         J --> L[Executar Busca e Salvar em -o];                            │
│     ----------                                                             │
│                                                                            │
│   ### 3.2. Key Components / Modules                                        │
│                                                                            │
│    Componente    | Localização    | Modificação P… | Responsabilid…        │
│   ---------------+----------------+----------------+----------------       │
│    rootCmd       |  cmd/perplexit | Adicionar      | Gerenciar o           │
│     Flags()      | y/root.go      |  flagFile  ( - | parsing da            │
│                  |                | f / --file ).  | nova flag de          │
│                  |                |                | input.                │
│    runQuery      |  cmd/perplexit | Refatorar a    | Orquestrar o          │
│                  | y/root.go      | lógica de      | novo fluxo de         │
│                  |                | obtenção da    | input da              │
│                  |                | query para     | query.                │
│                  |                | introduzir a   |                       │
│                  |                | leitura do     |                       │
│                  |                | arquivo        |                       │
│                  |                | ( flagFile )   |                       │
│                  |                | como           |                       │
│                  |                | prioridade.    |                       │
│    (Nova)        |  cmd/perplexit | Função         | Encapsular a          │
│     getQueryFrom | y/root.go      | auxiliar para  | lógica de I/O         │
│    File          |                | ler o conteúdo | e tratamento          │
│                  |                | do arquivo e   | de erro de            │
│                  |                | retornar o     | leitura de            │
│                  |                |  query  (e     | arquivo.              │
│                  |                |  error ).      |                       │
│                                                                            │
│   ### 3.3. Detailed Action Plan / Phases                                   │
│                                                                            │
│   #### Phase 1: Implementação da Flag de Input de Arquivo ( -f )           │
│                                                                            │
│   Objective(s): Adicionar e integrar a flag  -f / --file  para             │
│   carregar o                                                               │
│   query de um arquivo.                                                     │
│                                                                            │
│    Task              | Rationale/Goal    | Est… | Deliverable/Crit…        │
│   -------------------+-------------------+------+-------------------       │
│    1.1: Declarar     | Adicionar a       | S    |  var flagFile str        │
│    Flag              | variável global   |      | ing  em                  │
│                      |  flagFile  e      |      |  cmd/perplexity/r        │
│                      | registrá-la em    |      | oot.go ; Flag  -         │
│                      |  rootCmd.Flags()  |      | f / --file               │
│                      | .                 |      | registrada.              │
│    1.2: Criar        | Centralizar a     | S    | Nova função              │
│     getQueryFromFile | lógica de I/O.    |      |  getQueryFromFile        │
│                      |                   |      | (path string) (st        │
│                      |                   |      | ring, error)  em         │
│                      |                   |      |  cmd/perplexity/r        │
│                      |                   |      | oot.go .                 │
│                      |                   |      |                          │
│    1.3: Refatorar    | Mudar a ordem de  | M    |  runQuery                │
│     runQuery         | prioridade para a |      | verifica                 │
│                      | obtenção da       |      |  flagFile                │
│                      | query.            |      | primeiro; se             │
│                      |                   |      | presente, usa            │
│                      |                   |      |  getQueryFromFile        │
│                      |                   |      |   e anula                │
│                      |                   |      |  args / stdin  se        │
│                      |                   |      | bem-sucedido.            │
│    1.4: Teste        | Garantir que a    | S    | Novo teste em            │
│    Unitário          | nova função de    |      |  cmd/perplexity/r        │
│    (Simulado)        | obtenção de query |      | oot_test.go  (ou         │
│                      | funcione          |      | simulação de             │
│                      | corretamente.     |      | teste) para              │
│                      |                   |      | validar a                │
│                      |                   |      | precedência e            │
│                      |                   |      | leitura de               │
│                      |                   |      | arquivo.                 │
│                                                                            │
│   Exemplo de Lógica em  runQuery  (Task 1.3):                              │
│                                                                            │
│                                                                            │
│     ----------                                                             │
│     // cmd/perplexity/root.go - início de runQuery                         │
│     // ...                                                                 │
│     // 1. Obter query do arquivo se a flag -f estiver presente             │
│     if flagFile != "" {                                                    │
│         query, err = getQueryFromFile(flagFile)                            │
│         if err != nil {                                                    │
│             render.RenderError(err)                                        │
│             return err                                                     │
│         }                                                                  │
│     }                                                                      │
│                                                                            │
│     // 2. Se a query ainda estiver vazia, tentar args ou stdin             │
│     if query == "" {                                                       │
│         query, err = getQueryFromInput(args, os.Stdin, isTerminal)         │
│         if err != nil {                                                    │
│             // ...                                                         │
│         }                                                                  │
│     }                                                                      │
│     // ...                                                                 │
│     ----------                                                             │
│                                                                            │
│   #### Phase 2: Refinamento da Flag de Output de Arquivo ( -o )            │
│                                                                            │
│   Objective(s): Garantir que a flag de output existente esteja             │
│   corretamente mapeada e documentada.                                      │
│                                                                            │
│    Task              | Rationale/Goal     | E… | Deliverable/Crite…        │
│   -------------------+--------------------+----+--------------------       │
│    2.1: Verificar    | Confirmar o        | S  | Confirmação de            │
│    Declaração  -o    | mapeamento de  -   |    |  rootCmd.Flags().S        │
│                      | o / --output  para |    | tringVarP(&flagOut        │
│                      |  flagOutputFile .  |    | putFile, "output",        │
│                      |                    |    |  "o", "", "Save re        │
│                      |                    |    | sponse to file")          │
│                      |                    |    | ( $104  em                │
│                      |                    |    |  cmd/perplexity/ro        │
│                      |                    |    | ot.go ).                  │
│    2.2: Verificar    | Confirmar a lógica | S  | Confirmação de que        │
│    Uso da Output     | de salvamento      |    | a lógica é:               │
│                      | existente (já      |    |  if flagOutputFile        │
│                      | presente em  $307  |    |  != "" { os.WriteF        │
│                      | de                 |    | ile(flagOutputFile        │
│                      |  cmd/perplexity/ro |    | , ...)                    │
│                      | ot.go ) é mantida  |    |                           │
│                      | e trata            |    |                           │
│                      | corretamente os    |    |                           │
│                      | formatos  .md  e   |    |                           │
│                      |  .txt  (como       |    |                           │
│                      | simples salvamento |    |                           │
│                                                                            │
│   ### 3.4. API Design / Interface Changes                                  │
│                                                                            │
│   * Flags:                                                                 │
│     *  --output ,  -o : String. Caminho do arquivo para salvar a saída     │
│     (Markdown ou Text).                                                    │
│     *  --file ,  -f : String. Caminho do arquivo contendo o input da       │
│     query.                                                                 │
│   * Nova Função Interna:                                                   │
│                                                                            │
│     ----------                                                             │
│     // Exemplo de assinatura e corpo                                       │
│     func getQueryFromFile(path string) (string, error) {                   │
│         data, err := os.ReadFile(path)                                     │
│         if err != nil {                                                    │
│             return "", fmt.Errorf("failed to read input file %s: %w",      │
│   path, err)                                                               │
│         }                                                                  │
│         return strings.TrimSpace(string(data)), nil                        │
│     }                                                                      │
│     ----------                                                             │
│                                                                            │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 4. Key Considerations & Risk Mitigation                               │
│                                                                            │
│   ### 4.1. Technical Risks & Challenges                                    │
│                                                                            │
│   * Risco: Prioridade do Input: O risco de o input do arquivo ( -f )       │
│   ser ofuscado pelos argumentos ( args ) ou vice-versa.                    │
│     * Mitigação: Implementação da lógica de prioridade estrita em          │
│     runQuery :  -f  >  args  >  stdin . Onde o primeiro a fornecer uma     │
│     query não-vazia é usado.                                               │
│   * Risco: Arquivo Não Encontrado ( -f ): O usuário pode fornecer um       │
│   caminho de arquivo inválido.                                             │
│     * Mitigação:  getQueryFromFile  deve usar  os.ReadFile  e retornar     │
│     um  fmt.Errorf  claro se o arquivo não puder ser lido,                 │
│     interrompendo a execução com a mensagem de erro.                       │
│   * Risco: Query Vazia de Arquivo ( -f ): O arquivo pode existir, mas      │
│   estar vazio ou conter apenas espaços em branco.                          │
│     * Mitigação:  getQueryFromFile  deve usar  strings.TrimSpace  no       │
│     conteúdo lido e, se a query resultante for vazia,  runQuery  deve      │
│     mostrar a ajuda (ou retornar um erro específico, dependendo da         │
│     usabilidade desejada; a sugestão é cair na lógica de mostrar ajuda     │
│     se for a única fonte, mas aqui, se for a fonte explicitamente          │
│     escolhida, é melhor retornar um erro).                                 │
│                                                                            │
│                                                                            │
│   ### 4.2. Dependencies                                                    │
│                                                                            │
│   * Interna: Refatoração de  runQuery  e a nova função auxiliar            │
│   getQueryFromFile .                                                       │
│   * Externas: Uso de  os.ReadFile  e  strings.TrimSpace  (ambos são        │
│   pacotes padrão Go).                                                      │
│                                                                            │
│   ### 4.3. Non-Functional Requirements (NFRs) Addressed                    │
│                                                                            │
│   * Usabilidade: Aumenta a flexibilidade, permitindo queries complexas     │
│   em arquivos (útil para prompts longos) e facilitando a automação         │
│   (encadeamento de  perplexity  com outras ferramentas que geram           │
│   prompts).                                                                │
│   * Robustez: O tratamento explícito de erros de leitura de arquivo em     │
│   getQueryFromFile  torna o fluxo de input mais robusto.                   │
│   * Manutenibilidade: O encapsulamento da lógica de I/O em                 │
│   getQueryFromFile  mantém  runQuery  mais limpa e focada no controle      │
│   de fluxo.                                                                │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 5. Success Metrics / Validation Criteria                              │
│                                                                            │
│   * Input de Arquivo Funcional:  perplexity -f prompt.md  executa a        │
│   busca usando o conteúdo de  prompt.md  como query.                       │
│   * Precedência de Input:  perplexity "query de cli" -f prompt.md          │
│   deve executar a busca usando o conteúdo de  prompt.md  (flag  -f         │
│   tem precedência).                                                        │
│   * Output de Arquivo Funcional:  perplexity "quem descobriu o             │
│   brasil?" -o descobrimento.md  executa a busca e salva a resposta         │
│   completa em  descobrimento.md . O arquivo  descobrimento.md  deve        │
│   conter a resposta completa em formato Markdown (ou texto, dependendo     │
│   do renderizador).                                                        │
│   * Combinação:  perplexity -f prompt.md -o saida.txt  funciona            │
│   corretamente (Input de arquivo, Output para arquivo).                    │
│                                                                            │
│   --------                                                                 │
│                                                                            │
│   ## 6. Assumptions Made                                                   │
│                                                                            │
│   * O conteúdo do arquivo de input ( -f ) será tratado como texto          │
│   simples (não binário), e o  strings.TrimSpace  é desejado no             │
│   conteúdo lido.                                                           │
│   * A leitura de  flagFile  deve ter precedência sobre  args  ou           │
│   stdin .                                                                  │
│   * A flag  -o / --output  existente (que mapeia para  flagOutputFile      │
│   ) já é suficiente para salvar a saída do  responseText  em qualquer      │
│   arquivo especificado pelo usuário (seja  .md  ou  .txt ).                │
│                                                                            │
│   ## 7. Open Questions / Areas for Further Investigation                   │
│                                                                            │
│   * Deveria haver uma verificação de tamanho de arquivo de input ( -f      │
│   ) para evitar o carregamento de arquivos gigabytes na memória?           │
│   (Decisão: Não por agora, assumindo que os prompts de LLM são             │
│   razoavelmente pequenos, mas deve ser revisitado se houver problemas      │
│   de memória).                                                             │
│   * O que deve acontecer se a query for fornecida via  args  e  stdin      │
│   e  -f ? (Decisão: Manter a prioridade  [-f] > [args] > [stdin]  para     │
│   clareza).                                                                │
╰────────────────────────────────────────────────────────────────────────────╯