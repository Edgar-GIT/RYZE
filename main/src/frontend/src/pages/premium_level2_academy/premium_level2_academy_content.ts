import type { Locale } from "@/i18n/i18n";

// The Premium Level 2 academy reproduces the practical content of the official
// 8-week hypertrophy guide supplied with the service. The guide was written in
// Portuguese; every section is kept in both languages because the RYZE
// interface is bilingual. This is educational material, not a medical document
// and not the buyer's individual plan.

export interface BilingualText {
  en: string;
  pt: string;
}

export type AcademyBlock =
  | { kind: "paragraph"; text: BilingualText }
  | { kind: "list"; ordered?: boolean; items: BilingualText[] }
  | { kind: "cards"; items: { title: BilingualText; text: BilingualText }[] }
  | { kind: "table"; headers: BilingualText[]; rows: BilingualText[][] }
  | { kind: "callout"; tone?: "info" | "warning" | "success"; title?: BilingualText; text: BilingualText }
  | { kind: "references"; items: string[] };

export interface AcademyChapter {
  id: string;
  number: string;
  title: BilingualText;
  intro: BilingualText;
  blocks: AcademyBlock[];
}

export const academyText = (text: BilingualText, locale: Locale): string => text[locale];

const b = (en: string, pt: string): BilingualText => ({ en, pt });

export const ACADEMY_CHAPTERS: AcademyChapter[] = [
  {
    id: "start",
    number: "01",
    title: b("Start here", "Começa aqui"),
    intro: b(
      "Read once. Apply for eight weeks. Return to the right pages when you need them.",
      "Lê uma vez. Aplica durante oito semanas. Volta às páginas certas quando precisares."
    ),
    blocks: [
      {
        kind: "paragraph",
        text: b(
          "The goal is simple: give you a system for what to do inside and outside the gym. The programme tells you what to train. This guide teaches you how to train, how to progress and how to align nutrition with your goal.",
          "O objetivo é simples: dar-te um sistema para saberes o que fazer dentro e fora do ginásio. O programa diz-te o que treinar. Este guia ensina-te como treinar, como progredir e como alinhar a alimentação com o objetivo."
        )
      },
      {
        kind: "list",
        items: [
          b("Follow the programme: four sessions per week for eight weeks.", "Segue o programa: quatro treinos por semana durante oito semanas."),
          b("Record loads, repetitions and perception of effort.", "Regista cargas, repetições e sensação de esforço."),
          b("Progress: try to improve without sacrificing technique.", "Progride: tenta melhorar sem sacrificar a técnica."),
          b("Recover: sleep, food and rest are part of the plan.", "Recupera: sono, alimentação e descanso fazem parte do plano.")
        ]
      },
      {
        kind: "callout",
        tone: "warning",
        title: b("Important note", "Nota importante"),
        text: b(
          "This guide is educational and intended for healthy adults. Persistent pain, injury, pregnancy, illness, medication or clinical nutritional needs must be followed by qualified health professionals.",
          "Este guia é educativo e destina-se a adultos saudáveis. Dor persistente, lesão, gravidez, doença, medicação ou necessidades nutricionais clínicas devem ser acompanhadas por profissionais de saúde qualificados."
        )
      },
      {
        kind: "callout",
        tone: "info",
        title: b("Your individual plan comes from your trainer", "O teu plano individual vem do teu treinador"),
        text: b(
          "The 8-week guide describes the method used by the RYZE programmes. Your trainer reads your Premium Level 2 answers and adapts exercise selection, volume and nutrition to your body, experience, schedule and health.",
          "O guia de 8 semanas descreve o método usado pelos programas RYZE. O teu treinador lê as tuas respostas Premium Nível 2 e adapta a escolha de exercícios, o volume e a nutrição ao teu corpo, experiência, horário e saúde."
        )
      }
    ]
  },
  {
    id: "method",
    number: "02",
    title: b("The method", "O método"),
    intro: b(
      "Four steps, repeated week after week. You do not need a complicated method. You need to repeat the basics with quality.",
      "Quatro passos, repetidos semana após semana. Não precisas de um método complicado. Precisas de repetir o básico com qualidade."
    ),
    blocks: [
      {
        kind: "cards",
        items: [
          {
            title: b("Train", "Treina"),
            text: b("Complete the planned sets and approach failure with good technique.", "Cumpre as séries previstas e aproxima-te da falha com boa técnica.")
          },
          {
            title: b("Record", "Regista"),
            text: b("Write down weight and repetitions. Without a record, progression becomes random.", "Anota peso e repetições. Sem registo, a progressão fica ao acaso.")
          },
          {
            title: b("Progress", "Progride"),
            text: b("One more repetition or a small increase in load is already progress.", "Mais uma repetição ou um pequeno aumento de carga já é progresso.")
          },
          {
            title: b("Recover", "Recupera"),
            text: b("Sleep and nutrition turn the stimulus into adaptation.", "Sono e alimentação permitem transformar o estímulo em adaptação.")
          }
        ]
      },
      {
        kind: "callout",
        tone: "success",
        title: b("Programme rule", "Regra do programa"),
        text: b(
          "Do not chase feeling destroyed. Create a stimulus you can repeat and improve.",
          "Não procures ficar destruído. Procura criar um estímulo que consigas repetir e melhorar."
        )
      }
    ]
  },
  {
    id: "hypertrophy",
    number: "03",
    title: b("Hypertrophy", "Hipertrofia"),
    intro: b(
      "What actually makes a muscle grow: growth is the consequence of enough training plus enough recovery over time.",
      "O que realmente faz um músculo crescer: o crescimento é consequência de treino suficiente mais recuperação suficiente ao longo do tempo."
    ),
    blocks: [
      {
        kind: "cards",
        items: [
          { title: b("Stimulus", "Estímulo"), text: b("Sets with enough effort.", "Séries com esforço suficiente.") },
          { title: b("Recovery", "Recuperação"), text: b("Sleep and fatigue management.", "Sono e gestão da fadiga.") },
          { title: b("Progression", "Progressão"), text: b("More work over time.", "Mais trabalho ao longo do tempo.") },
          { title: b("Nutrition", "Nutrição"), text: b("Enough energy and protein.", "Energia e proteína adequadas.") },
          { title: b("Consistency", "Consistência"), text: b("Weeks and months, not days.", "Semanas e meses, não dias.") }
        ]
      },
      {
        kind: "paragraph",
        text: b(
          "Volume: as a general reference, around 10 weekly sets per muscle group is a useful starting point for hypertrophy, but it is not a fixed rule. The response depends on the muscle, experience, recovery and the quality of the sets.",
          "Volume: como referência geral, cerca de 10 séries semanais por grupo muscular é um ponto útil para hipertrofia, mas não é uma regra fixa. A resposta depende do músculo, experiência, recuperação e qualidade das séries."
        )
      },
      {
        kind: "callout",
        tone: "info",
        text: b(
          "More is not automatically better. Find the dose that lets you progress, recover and repeat good weeks.",
          "Mais não é automaticamente melhor. Procura a dose que te permita progredir, recuperar e repetir boas semanas."
        )
      }
    ]
  },
  {
    id: "progression",
    number: "04",
    title: b("Progression and intensity", "Progressão e intensidade"),
    intro: b(
      "Rep ranges, load selection, effort and rest. These are the tools that make progress measurable.",
      "Faixas de repetições, escolha de carga, esforço e descanso. Estas são as ferramentas que tornam o progresso mensurável."
    ),
    blocks: [
      {
        kind: "paragraph",
        text: b(
          "Stop chasing a fixed repetition number. A range gives you room to progress simply and objectively. Example: 3 sets of 8–10 repetitions. When you reach the top of the range with solid technique, increase the load slightly and return to the bottom of the range.",
          "Deixa de perseguir um número fixo de repetições. Uma faixa dá-te espaço para progredir de forma simples e objetiva. Exemplo: 3 séries de 8–10 repetições. Quando atinges o topo da faixa com técnica sólida, sobe ligeiramente a carga e volta à parte inferior da faixa."
        )
      },
      {
        kind: "table",
        headers: [b("Session", "Treino"), b("Load", "Carga"), b("Repetitions", "Repetições")],
        rows: [
          [b("1", "1"), b("80 kg", "80 kg"), b("8 / 8 / 7", "8 / 8 / 7")],
          [b("2", "2"), b("80 kg", "80 kg"), b("9 / 8 / 8", "9 / 8 / 8")],
          [b("3", "3"), b("80 kg", "80 kg"), b("10 / 10 / 9", "10 / 10 / 9")],
          [b("4", "4"), b("80 kg", "80 kg"), b("10 / 10 / 10", "10 / 10 / 10")],
          [b("5", "5"), b("85 kg", "85 kg"), b("8 / 8 / 7", "8 / 8 / 7")]
        ]
      },
      {
        kind: "paragraph",
        text: b(
          "Choosing the right load: the right load lets you complete the planned range with real effort and controlled technique.",
          "Como escolher a carga certa: a carga certa é aquela que te deixa cumprir a faixa prevista com esforço real e técnica controlada."
        )
      },
      {
        kind: "table",
        headers: [b("Too light", "Demasiado leve"), b("Right zone", "Zona certa"), b("Too heavy", "Demasiado pesado")],
        rows: [
          [
            b("You finish 10 reps and could still do 5 or 6. Increase the load.", "Terminas 10 reps e ainda conseguias fazer 5 ou 6. Aumenta a carga."),
            b("You finish the set and estimate 1–3 reps remained. Keep it and progress.", "Terminas a série e estimas que restavam 1 a 3 reps. Mantém e progride."),
            b("You cannot reach the minimum or technique collapses. Reduce the load.", "Não chegas ao mínimo da faixa ou a técnica colapsa. Reduz a carga.")
          ]
        ]
      },
      {
        kind: "paragraph",
        text: b(
          "Do not confuse load with ego. A heavier weight is only progress if you keep executing the movement within the defined standard. If you shorten the range, swing or need help, you are no longer comparing equivalent repetitions.",
          "Não confundas carga com ego. Um peso maior só é progresso se continuares a executar o movimento dentro do padrão definido. Se encurtas a amplitude, ganhas balanço ou precisas de ajuda, já não estás a comparar repetições equivalentes."
        )
      },
      {
        kind: "table",
        headers: [b("RIR", "RIR"), b("Meaning", "Significado")],
        rows: [
          [b("4", "4"), b("Comfortable", "Confortável")],
          [b("3", "3"), b("Moderate", "Moderado")],
          [b("2", "2"), b("Hard", "Difícil")],
          [b("1", "1"), b("Very hard", "Muito difícil")],
          [b("0", "0"), b("Failure", "Falha")]
        ]
      },
      {
        kind: "paragraph",
        text: b(
          "RIR means repetitions in reserve. On most sets of the programme, work at approximately 1–3 RIR. On safer isolation exercises, some sets may approach 0–1 RIR.",
          "RIR significa repetições em reserva. Na maior parte das séries do programa, trabalha aproximadamente entre 1 e 3 RIR. Em exercícios mais seguros e de isolamento, algumas séries podem aproximar-se de 0–1 RIR."
        )
      },
      {
        kind: "cards",
        items: [
          {
            title: b("Heavy compounds", "Compostos pesados"),
            text: b("Squat, RDL, bench press, rows: normally keep 1–3 RIR. Technical failure increases fatigue and can compromise later sets.", "Agachamento, RDL, supino, remadas: mantém normalmente 1–3 RIR. A falha técnica aumenta a fadiga e pode comprometer séries seguintes.")
          },
          {
            title: b("Machines and isolation", "Máquinas e isolamento"),
            text: b("Leg extension, lateral raises, curls, abduction: you can move closer to failure, especially on the last set, if technique remains controlled.", "Extensora, elevações laterais, curls, abdução: podes aproximar mais da falha, sobretudo na última série, se a técnica continuar controlada.")
          },
          {
            title: b("Every set to failure", "Todas as séries à falha"),
            text: b("Not necessary. More suffering does not automatically mean more growth. The stimulus must be recoverable.", "Não é necessário. Mais sofrimento não significa automaticamente mais crescimento. O estímulo deve ser recuperável.")
          }
        ]
      },
      {
        kind: "paragraph",
        text: b(
          "Rest enough to repeat quality. The rest period is not lost time; it is what lets you produce good sets. Compounds generally need 2–3 minutes; isolation work 1–2 minutes. If breathing is still uncontrolled and the next set will clearly drop in performance only because you rested too little, wait longer. For hypertrophy, rests above 60 seconds tend to be more favourable than very short intervals.",
          "Descansa o suficiente para repetir qualidade. O descanso não é tempo perdido; é o que te permite produzir boas séries. Os compostos precisam geralmente de 2–3 minutos; o isolamento, 1–2 minutos. Se a respiração ainda está descontrolada e sabes que a próxima série vai cair apenas porque descansaste pouco, espera mais. Para hipertrofia, descansos acima de 60 segundos tendem a ser mais favoráveis do que intervalos muito curtos."
        )
      },
      {
        kind: "paragraph",
        text: b(
          "Warm-up: prepare without spending the good sets. The purpose of a specific warm-up is to reach the first effective set ready, not tired. Example for an effective set with 80 kg: 40 kg × 8 (light), 55 kg × 5 (build), 70 kg × 2–3 (approach), then 80 kg × 8–10 as the working set. You do not need this many sets on every exercise. The heavier and more technical the first movement, the more useful the approach; later exercises for the same muscle usually need less.",
          "Aquecimento: prepara sem gastar as séries boas. O objetivo do aquecimento específico é chegar à primeira série efetiva pronto, não cansado. Exemplo para uma série efetiva com 80 kg: 40 kg × 8 (leve), 55 kg × 5 (subir carga), 70 kg × 2–3 (aproximação), depois 80 kg × 8–10 como série de trabalho. Não precisas deste número de séries em todos os exercícios. Quanto mais pesado e técnico for o primeiro movimento, mais útil é a aproximação; nos exercícios seguintes para o mesmo grupo muscular, normalmente precisas de menos."
        )
      },
      {
        kind: "paragraph",
        text: b(
          "Does the repetition count? Use four criteria: range (use the range you control without pain), control (avoid unnecessary swing and sudden drops), target (keep execution directed at the intended muscle group) and consistency (repeat the technique so you can compare week to week). You do not need to count seconds obsessively. Control the descent, avoid losing position and rise with intent. A deliberately slow repetition is not automatically superior.",
          "A repetição conta? Usa quatro critérios: amplitude (usa a amplitude que controlas e que o exercício permite sem dor), controlo (evita balanço desnecessário e quedas bruscas), alvo (mantém a execução orientada para o grupo muscular pretendido) e consistência (repete a técnica para conseguires comparar semana a semana). Não precisas de contar segundos obsessivamente. Controla a descida, evita perder a posição e sobe com intenção. Uma repetição deliberadamente lenta não é automaticamente superior."
        )
      },
      {
        kind: "paragraph",
        text: b(
          "Your record is your map. Without history, you do not know whether you are evolving or merely repeating the workout. Log date, exercise, load, sets, reps and RIR.",
          "O teu registo é o teu mapa. Sem histórico, não sabes se estás a evoluir ou apenas a repetir o treino. Regista data, exercício, carga, séries, repetições e RIR."
        )
      },
      {
        kind: "table",
        headers: [b("Date", "Data"), b("Exercise", "Exercício"), b("Load", "Carga"), b("S1", "S1"), b("S2", "S2"), b("S3", "S3"), b("RIR", "RIR")],
        rows: [
          [b("14/09", "14/09"), b("Hip thrust", "Hip thrust"), b("80 kg", "80 kg"), b("10", "10"), b("9", "9"), b("8", "8"), b("1–2", "1–2")],
          [b("18/09", "18/09"), b("Hip thrust", "Hip thrust"), b("80 kg", "80 kg"), b("10", "10"), b("10", "10"), b("9", "9"), b("1–2", "1–2")],
          [b("22/09", "22/09"), b("Hip thrust", "Hip thrust"), b("80 kg", "80 kg"), b("10", "10"), b("10", "10"), b("10", "10"), b("1", "1")],
          [b("26/09", "26/09"), b("Hip thrust", "Hip thrust"), b("85 kg", "85 kg"), b("8", "8"), b("8", "8"), b("7", "7"), b("1–2", "1–2")]
        ]
      },
      {
        kind: "list",
        items: [
          b("More repetitions with the same load and technique.", "Mais repetições com a mesma carga e técnica."),
          b("More load while staying in the rep range.", "Mais carga mantendo a faixa de repetições."),
          b("Same load and reps with better execution or more RIR.", "Mesma carga e repetições com melhor execução ou maior RIR."),
          b("More stability and range without losing performance.", "Mais estabilidade e amplitude sem perder rendimento.")
        ]
      },
      {
        kind: "paragraph",
        text: b(
          "When should you increase the load? Ask: are you inside the rep range with good technique? If not, keep or reduce the load. If yes, look at the top of the range. If you reached the top on the target sets, increase slightly and return to the lower part of the range. Small increases win. On machines or dumbbells, use the smallest available increment.",
          "Quando subir a carga? Pergunta: estás dentro da faixa com boa técnica? Se não, mantém ou reduz a carga. Se sim, olha para o topo da faixa. Se atingiste o topo nas séries-alvo, sobe ligeiramente e volta à zona inferior da faixa. Aumentos pequenos vencem. Em máquinas ou halteres, usa o incremento disponível mais pequeno."
        )
      },
      {
        kind: "paragraph",
        text: b(
          "If you stopped evolving, do not change everything. One worse week is not stagnation; look for a trend before changing the plan. Check: did you sleep and recover? Did you eat enough for the goal? Is technique still the same? Are you trying to progress too early? Have there been 2–3 weeks without any improvement? Consider a lighter week when performance keeps falling, fatigue is unusual, muscle soreness does not disappear, motivation is low and several sessions in a row are worse. A practical option is to temporarily reduce sets and/or load for a few days, then resume.",
          "Se deixaste de evoluir, não mudes tudo. Uma semana pior não é estagnação; procura uma tendência antes de mexer no plano. Verifica: dormiste e recuperaste? Comeste o suficiente para o objetivo? A técnica continua igual? Estás a tentar progredir cedo demais? Há 2–3 semanas sem qualquer melhoria? Considera uma semana mais leve quando o desempenho cai de forma persistente, a fadiga é fora do normal, as dores musculares não desaparecem, a motivação está baixa e várias sessões consecutivas são piores. Uma opção prática é reduzir temporariamente séries e/ou carga durante alguns dias e depois retomar."
        )
      },
      {
        kind: "cards",
        items: [
          {
            title: b("Rest-pause", "Rest-pause"),
            text: b("Do a hard set, rest 15–30 seconds and add a few controlled reps. Best on stable/isolation exercises.", "Faz uma série difícil, descansa 15–30 segundos e acrescenta poucas reps controladas. Bom em exercícios estáveis/de isolamento.")
          },
          {
            title: b("Drop set", "Drop set"),
            text: b("After the set, reduce the load and continue with controlled reps. Saves time; increases local fatigue.", "Após a série, reduz a carga e continua com reps controladas. Poupa tempo; aumenta a fadiga local.")
          },
          {
            title: b("Dead stop", "Dead stop"),
            text: b("A full pause between reps removes momentum. Helps standardise some executions.", "Pausa total entre repetições, retirando o embalo. Ajuda a padronizar algumas execuções.")
          },
          {
            title: b("Superset", "Supersérie"),
            text: b("Two exercises in sequence before resting. Useful for efficiency; combine exercises that do not interfere too much.", "Dois exercícios em sequência antes do descanso. Útil para eficiência; combina exercícios que não se atrapalhem demasiado.")
          }
        ]
      },
      {
        kind: "callout",
        tone: "info",
        text: b(
          "Priority: first improve technique, effort and progression. Advanced techniques are optional and never replace the basics done well.",
          "Prioridade: primeiro melhora a técnica, o esforço e a progressão. As técnicas avançadas são opcionais e não substituem o básico bem feito."
        )
      }
    ]
  },
  {
    id: "recovery",
    number: "05",
    title: b("Recovery", "Recuperação"),
    intro: b(
      "Training ends. Adaptation continues. Training more does not always mean evolving more; performance also depends on what happens between sessions.",
      "O treino acaba. A adaptação continua. Treinar mais nem sempre significa evoluir mais; o rendimento também depende do que acontece entre sessões."
    ),
    blocks: [
      {
        kind: "cards",
        items: [
          {
            title: b("7+ hours", "7+ horas"),
            text: b("Sleep: adults should regularly aim for at least seven hours; athletes may have individually higher needs.", "Sono: os adultos devem procurar regularmente pelo menos sete horas; atletas podem ter necessidades individuais maiores.")
          },
          {
            title: b("1–2 days", "1–2 dias"),
            text: b("Rest: distribute sessions so you do not repeatedly train muscles that are still very fatigued when that reduces quality.", "Descanso: distribui os treinos para não repetir músculos muito fatigados quando isso reduz a qualidade.")
          },
          {
            title: b("Performance signal", "Sinal de desempenho"),
            text: b("If loads and reps fall repeatedly, review sleep, diet, stress and volume.", "Se as cargas e as repetições caem repetidamente, revê sono, dieta, stresse e volume.")
          }
        ]
      },
      {
        kind: "callout",
        tone: "warning",
        text: b(
          "Muscle soreness does not measure quality. You can have an excellent session without being sore the next day. Use performance, technique and progression as more useful indicators.",
          "Dor muscular não mede qualidade. Podes ter um excelente treino sem ficar dorido no dia seguinte. Usa rendimento, técnica e progressão como indicadores mais úteis."
        )
      }
    ]
  },
  {
    id: "nutrition",
    number: "06",
    title: b("Nutrition", "Nutrição"),
    intro: b(
      "Training can be similar. Nutrition directs a large part of the body result.",
      "O treino pode ser semelhante. A alimentação é que direciona grande parte do resultado corporal."
    ),
    blocks: [
      {
        kind: "cards",
        items: [
          {
            title: b("Build muscle", "Ganhar músculo"),
            text: b("Slight energy surplus: more energy than you spend, without exaggerating. Fast weight gain tends to bring more fat.", "Ligeiro excedente energético: mais energia do que gastas, sem exagerar. Uma subida rápida de peso tende a trazer mais gordura.")
          },
          {
            title: b("Define", "Definir"),
            text: b("Moderate energy deficit: less energy than you spend, keeping protein and strength training to protect muscle mass.", "Défice energético moderado: menos energia do que gastas, mantendo proteína e treino de força para proteger massa muscular.")
          },
          {
            title: b("Recompose", "Recompor"),
            text: b("Maintenance or a small deficit. More likely in beginners, returns to training or people with more body fat. Requires patience.", "Manutenção ou pequeno défice. Mais provável em iniciantes, regressos ao treino ou pessoas com mais gordura. Exige paciência.")
          }
        ]
      },
      {
        kind: "callout",
        tone: "info",
        text: b(
          "Definition is not a repetition type. To reveal more definition you normally need to reduce body fat while keeping as much muscle mass as possible.",
          "Definição não é um tipo de repetição. Para revelar mais definição, normalmente é necessário reduzir gordura corporal enquanto se mantém o máximo de massa muscular possível."
        )
      },
      {
        kind: "paragraph",
        text: b(
          "Protein is a priority for preserving and building muscle. For strength training, a practical and robust range is approximately 1.6–2.2 g/kg/day. At 60 kg that is 96–132 g; at 70 kg, 112–154 g; at 80 kg, 128–176 g.",
          "A proteína é uma prioridade para preservar e construir músculo. Para quem treina força, uma faixa prática e robusta é aproximadamente 1,6–2,2 g/kg/dia. A 60 kg são 96–132 g; a 70 kg, 112–154 g; a 80 kg, 128–176 g."
        )
      },
      {
        kind: "list",
        items: [
          b("Include a protein source in 3–5 meals across the day.", "Inclui uma fonte de proteína em 3–5 refeições ao longo do dia."),
          b("Use foods you like and can repeat consistently.", "Usa alimentos que gostas e consegues repetir com consistência."),
          b("Whey is only a practical way to reach the total; it is not mandatory.", "Whey é apenas uma forma prática de atingir o total; não é obrigatório."),
          b("In a calorie deficit, prioritise the upper part of the range.", "Em défice calórico, faz sentido dar prioridade à parte alta da faixa.")
        ]
      },
      {
        kind: "paragraph",
        text: b(
          "Carbohydrates provide energy; fats also have a place. After enough protein, distribute the remaining calories in a way that supports training, satiety and your preferences. Carbohydrates: rice, pasta, potato, bread, oats, fruit, cereals — useful for performance and recovery. Fats: olive oil, nuts, eggs, avocado, oily fish — they do not need to be eliminated.",
          "Os hidratos fornecem energia; as gorduras também têm lugar. Depois de garantires proteína suficiente, distribui o resto das calorias de forma a sustentar o treino, a saciedade e as tuas preferências. Hidratos: arroz, massa, batata, pão, aveia, fruta, cereais — úteis para rendimento e recuperação. Gorduras: azeite, frutos secos, ovos, abacate, peixe gordo — não precisam de ser eliminadas."
        )
      },
      {
        kind: "callout",
        tone: "info",
        title: b("A simple plate", "Um prato simples"),
        text: b(
          "Protein + a carbohydrate source + fruit/vegetables + some fat according to calories. Avoid extremes. Diets that remove food groups without need make adherence harder and do not replace control of total energy.",
          "Proteína + fonte de hidratos + fruta/vegetais + alguma gordura, conforme as calorias. Evita extremos. Dietas que retiram grupos alimentares sem necessidade tornam a adesão mais difícil e não substituem o controlo da energia total."
        )
      },
      {
        kind: "list",
        ordered: true,
        items: [
          b("Choose the direction: grow with a slight surplus; define with a moderate deficit.", "Escolhe a direção: crescer com um ligeiro excedente; definir com um défice moderado."),
          b("Observe 2–3 weeks: average weight, measurements, photos, hunger and performance.", "Observa 2–3 semanas: peso médio, medidas, fotos, fome e desempenho."),
          b("Make a small adjustment: if there is no trend, change 100–200 kcal and observe again.", "Faz um pequeno ajuste: se não há tendência, altera 100–200 kcal e volta a observar.")
        ]
      },
      {
        kind: "paragraph",
        text: b(
          "To gain muscle, avoid an aggressive bulk; larger surpluses tend to increase mostly fat gain. To define, very large deficits make recovery harder and can harm lean-mass gains; a meta-regression found that a deficit of around 500 kcal/day was associated, on average, with an absence of lean-mass gains.",
          "Para ganhar músculo, evita um bulk agressivo; excedentes maiores tendem a aumentar sobretudo o ganho de gordura. Para definir, défices muito grandes dificultam a recuperação e podem prejudicar ganhos de massa magra; uma meta-regressão encontrou que um défice de cerca de 500 kcal/dia se associou, em média, à ausência de ganhos de massa magra."
        )
      },
      {
        kind: "paragraph",
        text: b(
          "Before and after training: the daily total matters more than chasing a window of minutes. Two to three hours before, choose a meal you tolerate well with carbohydrates and protein; the closer to training, the simpler and easier to digest it should be. Afterwards, eat a normal meal with protein and carbohydrates and keep meeting the daily total. You do not need to run for a shake at 30 minutes. Examples: yoghurt + banana + cereal, or bread + eggs + fruit before; rice + chicken + vegetables, or yoghurt + fruit + oats after. Training fasted can work for some people but is not mandatory or superior for definition; choose what lets you train well and follow the diet.",
          "Antes e depois do treino: o total do dia é mais importante do que perseguir uma janela de minutos. Duas a três horas antes, escolhe uma refeição que toleres bem, com hidratos e proteína; quanto mais perto do treino, mais simples e fácil de digerir deve ser. Depois, faz uma refeição normal com proteína e hidratos e continua a cumprir o total diário. Não precisas de correr para um batido aos 30 minutos. Exemplos: iogurte + banana + cereais, ou pão + ovos + fruta antes; arroz + frango + legumes, ou iogurte + fruta + aveia depois. Treinar em jejum pode funcionar para algumas pessoas, mas não é obrigatório nem superior para definição; escolhe o que te permite treinar melhor e cumprir a dieta."
        )
      },
      {
        kind: "paragraph",
        text: b(
          "Hydration and food quality: performance can worsen when hydration and basic nutrition are ignored. Drink water regularly and reinforce hydration on hot days or sessions with a lot of sweat. Include a variety of fruit and vegetables daily for fibre, micronutrients and satiety. Increase fibre gradually; too much fibre immediately before training can cause discomfort. You do not need to avoid salt extremely if you are healthy and sweat; adapt to context and clinical guidance.",
          "Hidratação e qualidade alimentar: o desempenho pode piorar quando a hidratação e a alimentação básica são ignoradas. Bebe água ao longo do dia e reforça a hidratação em dias quentes ou sessões com muito suor. Inclui variedade de fruta e vegetais todos os dias para fibra, micronutrientes e saciedade. Aumenta a fibra gradualmente; demasiada fibra imediatamente antes do treino pode causar desconforto. Não precisas de evitar sal de forma extrema se és saudável e transpiras; adapta ao contexto e à orientação clínica."
        )
      },
      {
        kind: "cards",
        items: [
          {
            title: b("Creatine monohydrate", "Creatina monohidratada"),
            text: b("3–5 g/day. May help performance in intense efforts and training adaptations. Take daily; no cycling needed.", "3–5 g/dia. Pode ajudar o desempenho em esforços intensos e as adaptações ao treino. Toma diariamente; não precisa de ciclo.")
          },
          {
            title: b("Whey / protein", "Whey / proteína"),
            text: b("Convenience. Not superior to an adequate diet; it makes reaching daily protein easier.", "Conveniência. Não é superior a uma dieta adequada; torna mais fácil atingir a proteína diária.")
          },
          {
            title: b("Caffeine", "Cafeína"),
            text: b("Optional. May improve performance; response varies. Doses of 3–6 mg/kg have good evidence, but use the lowest effective dose and consider side effects and sleep.", "Opcional. Pode melhorar o desempenho; a resposta varia. Doses de 3–6 mg/kg têm boa evidência, mas usa a menor dose eficaz e considera os efeitos secundários e o impacto no sono.")
          }
        ]
      },
      {
        kind: "callout",
        tone: "warning",
        text: b(
          "Caffeine is not mandatory and can worsen sleep, anxiety or palpitations. Minors, pregnant people and people with clinical conditions or medication should seek professional guidance before using supplements or stimulants.",
          "A cafeína não é obrigatória e pode piorar o sono, a ansiedade ou palpitações. Menores, grávidas e pessoas com condições clínicas ou medicação devem procurar orientação profissional antes de usar suplementos ou estimulantes."
        )
      },
      {
        kind: "paragraph",
        text: b(
          "Cardio is a tool, not punishment. Use it for health, physical capacity and to help create a deficit without turning strength training into a secondary activity. Start with daily steps before adding much cardio; add 1–3 light/moderate sessions if you need more expenditure; monitor whether legs and loads worsen and review volume, intensity or timing. Use the minimum effective dose: enough to help the goal without destroying leg recovery. Preserve the quality of strength training; if cardio stops you progressing, the problem is recovery management, not lack of willpower.",
          "O cardio é uma ferramenta, não um castigo. Usa-o para saúde, capacidade física e para ajudar a criar défice, sem transformar o treino de força em segundo plano. Começa pelos passos diários antes de acrescentar muito cardio; adiciona 1–3 sessões leves/moderadas se precisares de mais gasto; monitoriza se as pernas e as cargas pioram e revê volume, intensidade ou timing. Usa a dose mínima eficaz: o suficiente para ajudar o objetivo sem destruir a recuperação das pernas. Preserva a qualidade do treino de força; se o cardio te impede de progredir, o problema é gestão de recuperação, não falta de vontade."
        )
      }
    ]
  },
  {
    id: "cases",
    number: "07",
    title: b("Practical cases", "Exemplos práticos"),
    intro: b(
      "Three applications of the same principles. They are examples, not personalised diets.",
      "Três aplicações dos mesmos princípios. São exemplos, não dietas personalizadas."
    ),
    blocks: [
      {
        kind: "cards",
        items: [
          {
            title: b("Case A — woman, 60 kg, build muscle", "Caso A — mulher, 60 kg, ganhar massa"),
            text: b("96–132 g protein/day, 4 sessions/week, small calorie surplus. Follow the four days of the female programme and record loads. Add reps before load. Include protein in every main meal. Aim for a slow weight rise, not eating everything. If 2–3 weeks pass without a trend while training is consistent, increase calories slightly. Example day: yoghurt + oats + fruit; rice + chicken + vegetables; bread + fresh cheese or yoghurt; potato + fish + salad. Quantities depend on individual needs.", "96–132 g de proteína/dia, 4 treinos/semana, pequeno excedente calórico. Cumpre os quatro dias do programa feminino e regista cargas. Acrescenta repetições antes de subir carga. Inclui proteína em todas as refeições principais. Procura uma subida lenta de peso, sem comer tudo. Se passarem 2–3 semanas sem tendência e o treino estiver consistente, aumenta ligeiramente as calorias. Exemplo de dia: iogurte + aveia + fruta; arroz + frango + legumes; pão + queijo fresco ou iogurte; batata + peixe + salada. As quantidades dependem das necessidades individuais.")
          },
          {
            title: b("Case B — man, 80 kg, build muscle", "Caso B — homem, 80 kg, ganhar massa"),
            text: b("128–176 g protein/day, 4 sessions/week, small surplus. Do not add random chest/back/shoulder sets outside the programme. Treat legs as part of the goal, not optional. Keep 1–3 RIR on compounds and move closer to failure on safe isolation. Use a small surplus, enough protein and carbohydrates for performance. If the waist rises quickly and performance does not improve, review the surplus. The goal is not gaining weight at any cost: use photos, measurements, loads and reps as well as body weight.", "128–176 g de proteína/dia, 4 treinos/semana, pequeno excedente. Não acrescentes séries aleatórias de peito/costas/ombros fora do programa. Trata as pernas como parte real do objetivo, não como treino opcional. Mantém 1–3 RIR nos compostos e aproxima-te mais da falha em isolamentos seguros. Usa um excedente pequeno, proteína adequada e hidratos para suportar o desempenho. Se a cintura sobe rapidamente e o desempenho não melhora, revê o excedente. O objetivo não é ganhar peso a qualquer custo: usa fotos, medidas, cargas e repetições além do peso corporal.")
          },
          {
            title: b("Case C — 70 kg, define while keeping muscle", "Caso C — 70 kg, definir mantendo músculo"),
            text: b("112–154 g protein/day, 4 sessions/week, moderate deficit. Try to maintain strength and reps as much as possible. Use the upper protein range, especially if the deficit is demanding. Start cardio with daily activity and add only what is needed. Do not make the deficit so aggressive that it destroys training, sleep and hunger. Assess the weekly average weight; do not react to one-day fluctuations. Warning sign: fast weight loss + clear performance drop + extreme hunger + worse recovery means you are probably forcing too much. Adjust before simply enduring.", "112–154 g de proteína/dia, 4 treinos/semana, défice moderado. Tenta manter força e repetições o máximo possível. Usa a parte alta da proteína, especialmente se o défice for mais exigente. Começa o cardio pela atividade diária e adiciona apenas o necessário. Não tornes o défice tão agressivo que destrua treino, sono e fome. Avalia a média semanal do peso; não reajas a flutuações de um único dia. Sinal de alerta: perda rápida de peso + quebra clara de desempenho + fome extrema + recuperação pior significa que provavelmente estás a forçar demasiado. Ajusta antes de simplesmente aguentar.")
          }
        ]
      },
      {
        kind: "callout",
        tone: "warning",
        text: b(
          "These examples illustrate the method. Your trainer adjusts quantities, food choices and training to your intake, health and routine.",
          "Estes exemplos ilustram o método. O teu treinador ajusta quantidades, escolhas alimentares e treino ao teu questionário, saúde e rotina."
        )
      }
    ]
  },
  {
    id: "mistakes",
    number: "08",
    title: b("Mistakes that slow results", "Erros que travam resultados"),
    intro: b(
      "Before blaming the programme, confirm these basics are actually happening.",
      "Antes de culpar o programa, confirma se estas bases estão realmente a acontecer."
    ),
    blocks: [
      {
        kind: "cards",
        items: [
          { title: b("Changing exercises every week", "Trocar exercícios todas as semanas"), text: b("You never give yourself time to master and compare.", "Não dás tempo para dominar e comparar.") },
          { title: b("Always training far from failure", "Treinar sempre longe da falha"), text: b("The set feels tiring, but the stimulus may be low.", "A série parece cansativa, mas o estímulo pode ser baixo.") },
          { title: b("Failing everything", "Falhar tudo"), text: b("Fatigue rises faster than quality.", "A fadiga sobe mais rápido do que a qualidade.") },
          { title: b("Not recording", "Não registar"), text: b("You depend on memory and feeling.", "Ficas dependente da memória e da sensação.") },
          { title: b("Cutting too many calories", "Cortar demasiadas calorias"), text: b("Recovery and lean mass pay the bill.", "A recuperação e a massa magra pagam a conta.") },
          { title: b("Sleeping too little", "Dormir pouco"), text: b("Training, hunger and recovery become harder.", "Treino, fome e recuperação tornam-se mais difíceis.") },
          { title: b("Copying someone else's diet", "Copiar a dieta de outra pessoa"), text: b("Goals, weight and routine are different.", "Objetivos, peso e rotina são diferentes.") },
          { title: b("Wanting to change in 7 days", "Querer mudar em 7 dias"), text: b("Hypertrophy is a slow process.", "Hipertrofia é um processo lento.") }
        ]
      }
    ]
  },
  {
    id: "tracker",
    number: "09",
    title: b("The 8-week tracker", "O acompanhamento de 8 semanas"),
    intro: b(
      "Track execution, not only the mirror. Mark every week; the goal is to accumulate good weeks.",
      "Acompanha a execução, não só o espelho. Marca cada semana; o objetivo é acumular semanas boas."
    ),
    blocks: [
      {
        kind: "table",
        headers: [b("Week", "Semana"), b("4 sessions", "4 treinos"), b("Recorded", "Registei"), b("Protein", "Proteína"), b("Sleep", "Sono"), b("Note", "Nota")],
        rows: Array.from({ length: 8 }, (_, index) => [
          b(String(index + 1), String(index + 1)),
          b("□", "□"),
          b("□", "□"),
          b("□", "□"),
          b("□", "□"),
          b("", "")
        ])
      },
      {
        kind: "callout",
        tone: "success",
        text: b(
          "Final goal: you do not need eight perfect weeks. You need enough consistency for small progress to accumulate.",
          "Objetivo final: não precisas de oito semanas perfeitas. Precisas de consistência suficiente para que os pequenos progressos se acumulem."
        )
      }
    ]
  },
  {
    id: "rules",
    number: "10",
    title: b("The 10 rules of the method", "As 10 regras do método"),
    intro: b(
      "If you forget everything else, remember this.",
      "Se te esqueceres de tudo, lembra-te disto."
    ),
    blocks: [
      {
        kind: "list",
        ordered: true,
        items: [
          b("Follow the plan before trying to improve it.", "Cumpre o plano antes de o tentares melhorar."),
          b("Record loads and repetitions.", "Regista cargas e repetições."),
          b("Use rep ranges and double progression.", "Usa rep ranges e progressão dupla."),
          b("Train most sets close to failure, not necessarily to failure.", "Treina a maioria das séries perto da falha, não necessariamente até à falha."),
          b("Rest enough to repeat quality.", "Descansa o suficiente para repetir qualidade."),
          b("Consistent technique makes progression measurable.", "Técnica consistente torna a progressão mensurável."),
          b("Daily protein: approximately 1.6–2.2 g/kg.", "Proteína diária: aproximadamente 1,6–2,2 g/kg."),
          b("Adjust calories to the goal and to the trend observed over 2–3 weeks.", "Ajusta as calorias ao objetivo e à tendência observada ao longo de 2–3 semanas."),
          b("Sleep and recovery are part of the programme.", "Sono e recuperação fazem parte do programa."),
          b("Real results are built over consistent weeks.", "Resultados reais são construídos por semanas consistentes.")
        ]
      },
      {
        kind: "callout",
        tone: "success",
        text: b(
          "Do not seek the perfect workout. Seek to improve what you can repeat.",
          "Não procures o treino perfeito. Procura melhorar aquilo que consegues repetir."
        )
      }
    ]
  },
  {
    id: "science",
    number: "11",
    title: b("What the evidence reinforces", "O que a evidência reforça"),
    intro: b(
      "Recent principles that help apply the method with more precision. The guide translates evidence into practical rules; it does not replace individual assessment.",
      "Princípios recentes que ajudam a aplicar o método com mais precisão. O guia traduz a evidência para regras práticas; não substitui avaliação individual."
    ),
    blocks: [
      {
        kind: "cards",
        items: [
          {
            title: b("Load and repetitions", "Carga e repetições"),
            text: b("8–12 reps is not a magic range. A 2025 study compared higher loads (8–12 reps) with lower loads (20–25 reps) taken to voluntary fatigue and found similar hypertrophy in upper and lower limbs. Higher loads remain more specific for maximal strength. Keep the planned rep ranges because they make technique, control and progression easier; you do not need a magic number, you need enough effort, good execution and progression over time.", "8–12 repetições não é uma faixa mágica. Um estudo de 2025 comparou cargas mais altas (8–12 repetições) com cargas mais baixas (20–25 repetições), levadas à fadiga voluntária, e encontrou hipertrofia semelhante nos membros superiores e inferiores. Cargas mais altas continuam a ser mais específicas para desenvolver força máxima. Mantém as faixas definidas porque facilitam técnica, controlo e progressão; não precisas de um número mágico, precisas de esforço suficiente, boa execução e progressão ao longo do tempo.")
          },
          {
            title: b("Volume and frequency", "Volume e frequência"),
            text: b("More volume can help, with diminishing returns. A 2026 meta-regression with 67 studies and 2,058 participants found a positive relationship between weekly volume and hypertrophy, but with diminishing returns. After adjusting volume, increasing frequency by itself showed a small or negligible effect. Do not add sets at random. If you are progressing and recovering, keep it. Use more days mainly to distribute weekly volume, preserve set quality and manage fatigue.", "Mais volume pode ajudar, com retornos decrescentes. Uma meta-regressão de 2026 com 67 estudos e 2 058 participantes encontrou uma relação positiva entre volume semanal e hipertrofia, mas com retornos decrescentes. Depois de ajustado o volume, aumentar a frequência por si só mostrou um efeito pequeno ou negligenciável. Não acrescentes séries ao acaso. Se estás a progredir e a recuperar, mantém. Usa mais dias sobretudo para distribuir o volume semanal, preservar a qualidade das séries e gerir a fadiga.")
          },
          {
            title: b("Quadriceps", "Quadríceps"),
            text: b("Squat and leg extension are not redundant. In a 2026 randomised trial with 63 untrained women, leg extension produced greater rectus femoris growth in all three measured regions, while Smith machine squat produced greater distal vastus lateralis growth. The squat also improved squat strength more. Choose by target, comfort and ability to progress.", "Agachamento e extensão de joelho não são redundantes. Num ensaio randomizado de 2026 com 63 mulheres não treinadas, a extensão de joelho promoveu maior crescimento do reto femoral nas três regiões medidas, enquanto o agachamento na Smith promoveu maior crescimento distal do vasto lateral. O agachamento também melhorou mais a força no próprio agachamento. Escolhe pelo alvo, conforto e capacidade de progredir.")
          },
          {
            title: b("Unilateral vs bilateral", "Unilateral vs bilateral"),
            text: b("A 2025 meta-analysis found no significant difference in hypertrophy between unilateral and bilateral training. Strength was specific: bilateral training improved bilateral strength more; unilateral training improved unilateral strength more. Use squats, leg press, lunges or split squats according to goal, comfort, stability and ability to progress.", "Uma meta-análise de 2025 não encontrou diferença significativa de hipertrofia entre treino unilateral e bilateral. A força mostrou especificidade: treinar bilateralmente melhorou mais a força bilateral; treinar unilateralmente melhorou mais a força unilateral. Usa agachamentos, leg press, afundos ou split squats de acordo com o objetivo, conforto, estabilidade e capacidade de progredir.")
          },
          {
            title: b("Range and muscle length", "Amplitude e comprimento muscular"),
            text: b("Longer positions are promising, not magic. A 2025 meta-analysis found small average differences between training at longer and shorter muscle lengths, although regional trends appeared. Current evidence does not justify saying lengthened partials are universally superior. Use a full, controlled range when well tolerated; exercises that challenge the muscle in lengthened positions can help, but are not mandatory or superior in every context.", "Posições mais alongadas são promissoras, não são magia. Uma meta-análise de 2025 encontrou diferenças médias pequenas entre treinar em comprimentos musculares mais longos e mais curtos, embora tenham surgido tendências regionais. A evidência atual não justifica afirmar que parciais alongadas são universalmente superiores. Usa amplitude completa e controlada quando for bem tolerada; exercícios que desafiam o músculo em posições alongadas podem ser úteis, mas não são obrigatórios nem superiores em todos os contextos.")
          },
          {
            title: b("Biceps: no magic curl", "Bíceps: não existe um curl mágico"),
            text: b("In a 2025 randomised trial, 63 women trained for eight weeks. Incline curl produced greater thickness increase in the proximal region of the elbow flexors, while preacher curl produced greater increase in the distal region. Strength also improved more in the specifically trained exercise. Shoulder position alone does not tell the whole story: another 2025 trial found preacher curl and Bayesian cable curl increased biceps and brachialis thickness similarly across regions. Choose variations you can perform with stability, comfortable range and measurable progression.", "Num ensaio randomizado de 2025, 63 mulheres treinaram durante oito semanas. O incline curl produziu maior aumento da espessura na região proximal dos flexores do cotovelo, enquanto o preacher curl produziu maior aumento na região distal. A força também melhorou mais no exercício especificamente treinado. A posição do ombro, sozinha, não conta a história toda: outro ensaio de 2025 mostrou que preacher curl e Bayesian cable curl aumentaram de forma semelhante a espessura do bíceps e do braquial em várias regiões. Escolhe variações que consigas executar com estabilidade, amplitude confortável e progressão mensurável.")
          }
        ]
      },
      {
        kind: "callout",
        tone: "info",
        text: b(
          "Exercise rule: choose exercises by target muscle, tolerated range, stability, possibility of progression and absence of relevant pain. Use the science to choose good options, but keep exercises long enough to learn, record and progress.",
          "Regra: escolhe exercícios pelo músculo-alvo, amplitude tolerada, estabilidade, possibilidade de progressão e ausência de dor relevante. Usa a ciência para escolher boas opções, mas mantém os exercícios tempo suficiente para aprender, registar e progredir."
        )
      },
      {
        kind: "references",
        items: [
          "Currier BS, D'Souza AC, Fiatarone Singh MA, et al. American College of Sports Medicine Position Stand. Resistance Training Prescription for Muscle Function, Hypertrophy, and Physical Performance in Healthy Adults: An Overview of Reviews. Med Sci Sports Exerc. 2026;58(4):851-872. doi:10.1249/MSS.0000000000003897. PMID: 41843416.",
          "Robinson ZP, Pelland JC, Remmert JF, et al. Exploring the Dose-Response Relationship Between Estimated Resistance Training Proximity to Failure, Strength Gain, and Muscle Hypertrophy: A Series of Meta-Regressions. Sports Med. 2024;54(9):2209-2231. doi:10.1007/s40279-024-02069-2. PMID: 38970765.",
          "Grgic J, Schoenfeld BJ, Orazem J, Sabol F. Effects of resistance training performed to repetition failure or non-failure on muscular strength and hypertrophy: a systematic review and meta-analysis. J Sport Health Sci. 2022;11(2):202-211. doi:10.1016/j.jshs.2021.01.007. PMID: 33497853.",
          "Singer A, et al. Give it a rest: a systematic review with Bayesian meta-analysis on the effect of inter-set rest interval duration on muscle hypertrophy. Front Sports Act Living. 2024;6:1429789. doi:10.3389/fspor.2024.1429789. PMID: 39205815.",
          "Schoenfeld BJ, Grgic J, Krieger J. How many times per week should a muscle be trained to maximize muscle hypertrophy? A systematic review and meta-analysis. J Sports Sci. 2019;37(11):1286-1295. doi:10.1080/02640414.2018.1555906. PMID: 30558493.",
          "Morton RW, et al. A systematic review, meta-analysis and meta-regression of the effect of protein supplementation on resistance training-induced gains in muscle mass and strength. Br J Sports Med. 2018;52(6):376-384. doi:10.1136/bjsports-2017-097608. PMID: 28698222.",
          "Jäger R, et al. International Society of Sports Nutrition Position Stand: protein and exercise. J Int Soc Sports Nutr. 2017;14:20. doi:10.1186/s12970-017-0177-8. PMID: 28642676.",
          "Murphy C, Koehler K. Energy deficiency impairs resistance training gains in lean mass but not strength: a meta-analysis and meta-regression. Scand J Med Sci Sports. 2022;32(1):125-137. doi:10.1111/sms.14075. PMID: 34623696.",
          "Helms ER, et al. Effect of Small and Large Energy Surpluses on Strength, Muscle, and Skinfold Thickness in Resistance-Trained Individuals: A Parallel Groups Design. Sports Med Open. 2023;9(1):102. doi:10.1186/s40798-023-00651-y. PMID: 37914977.",
          "Kreider RB, et al. International Society of Sports Nutrition position stand: safety and efficacy of creatine supplementation in exercise, sport, and medicine. J Int Soc Sports Nutr. 2017;14:18. doi:10.1186/s12970-017-0173-z. PMID: 28615996.",
          "Guest NS, et al. International society of sports nutrition position stand: caffeine and exercise performance. J Int Soc Sports Nutr. 2021;18:1. doi:10.1186/s12970-020-00383-4. PMID: 33388079.",
          "Walsh NP, et al. Sleep and the athlete: narrative review and 2021 expert consensus recommendations. Br J Sports Med. 2021;55(7):356-368. doi:10.1136/bjsports-2020-102025. PMID: 33144349.",
          "Androulakis-Korakakis P, et al. Optimizing Resistance Training Technique to Maximize Muscle Hypertrophy: A Narrative Review. J Funct Morphol Kinesiol. 2024;9(1):9. doi:10.3390/jfmk9010009. PMID: 38249086.",
          "Carvalho L, et al. Muscle hypertrophy and strength gains after resistance training with different volume-matched loads: a systematic review and meta-analysis. Appl Physiol Nutr Metab. 2022;47(4):357-368. doi:10.1139/apnm-2021-0515. PMID: 35015560.",
          "Pelland JC, Remmert JF, Robinson ZP, Hinson SR, Zourdos MC. The Resistance Training Dose Response: Meta-Regressions Exploring the Effects of Weekly Volume and Frequency on Muscle Hypertrophy and Strength Gains. Sports Med. 2026;56(2):481-505. doi:10.1007/s40279-025-02344-w. PMID: 41343037.",
          "Lees MJ, McLeod JC, Morton RW, et al. Resistance training load does not determine resistance training-induced hypertrophy across upper and lower limbs in healthy young males. J Physiol. 2026. doi:10.1113/JP289684. PMID: 41474371.",
          "Kassiano W, Costa B, Kunevaliki G, et al. Comparison of Muscle Hypertrophy and Strength Adaptations Induced by Back Squat and Leg Extension Resistance Exercises. J Strength Cond Res. 2026;40(4):367-376. doi:10.1519/JSC.0000000000005338. PMID: 41379528.",
          "Kassiano W, Nunes JP, Costa B, Ribeiro AS, Loenneke JP, Cyrino ES. Comparison of Muscle Growth and Dynamic Strength Adaptations Induced by Unilateral and Bilateral Resistance Training: A Systematic Review and Meta-analysis. Sports Med. 2025;55(4):923-936. doi:10.1007/s40279-024-02169-z. PMID: 39794667.",
          "Varovic D, Wolf M, Schoenfeld BJ, Steele J, Grgic J, Mikulic P. Does Muscle Length Influence Regional Hypertrophy? A Systematic Review and Meta-Analysis. Int J Sports Med. 2025;46(14):1027-1036. doi:10.1055/a-2615-4935. PMID: 40570881.",
          "Kassiano W, Costa B, Kunevaliki G, et al. Distinct muscle growth and strength adaptations after preacher and incline biceps curls. Int J Sports Med. 2025;46(5):334-343. doi:10.1055/a-2517-0509. PMID: 39809454.",
          "Attarieh P, Nunes JP, Khani S, et al. Comparison Between Shoulder Flexed and Extended Positions in Elbow Flexion Resistance Training on Regional Hypertrophy and Maximum Strength: Preacher versus Bayesian Cable Curls. Eur J Sport Sci. 2025;25(4):e12279. doi:10.1002/ejsc.12279. PMID: 40082069."
        ]
      }
    ]
  },
  {
    id: "closing",
    number: "12",
    title: b("Train with intention", "Treina com intenção"),
    intro: b(
      "Progress with method. Evolve with consistency.",
      "Progride com método. Evolui com consistência."
    ),
    blocks: [
      {
        kind: "paragraph",
        text: b(
          "Hypertrophy & Definition — Female and Male Programmes · 8 weeks. Educational content for healthy adults. It does not replace individual medical or nutritional assessment.",
          "Hipertrofia & Definição — Programas Feminino e Masculino · 8 semanas. Conteúdo educativo para adultos saudáveis. Não substitui avaliação médica ou nutricional individual."
        )
      }
    ]
  }
];
