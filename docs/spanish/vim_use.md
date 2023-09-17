# How do I use VIM?

Este entorno usa [LunarVim](https://www.lunarvim.org/), por lo cual el
comando editor es `lvim`.

Cuando empece en VIM sin conocimientos para moverme rapidamente en el
editor no lo pense mucho, dejaba presionado `j` o `k` para moverme arriba
o abajo en mi archivo, `h` y `j` para moverme en la linea... con el tiempo
esto se hizo demasiado molesto asi que busque alternativas, empece a buscar
como moverme rapidamente en VIM... esta en mi forma actual.

---

1. **[Disclaimer](#disclaimer)**
2. **[Key leader](#key-leader)**
3. **[Fast moving in the file](#fast-moving-in-the-file)**

   - 3.1 **[Actions in file](#actions-in-file)**

4. **[Fast moving in the line](#fast-moving-in-the-line)**

   - 4.1 **[Actions in line](#actions-in-line)**

5. **[Other common actions](#other-common-actions)**
6. **[File navigation](#file-navigation)**
7. **[Tricks with Tabs and splits](#tricks-with-tabs-and-splits)**
8. **[Replace text](#replace-text)**
9. **[Macros](#macros)**
10. **[Records](#records)**
11. **[Indent lines](#indent-lines)**
12. **[Suspend Vim](#suspend-vim)**
13. **[:fire: Programming in Vim :fire:](#-programming-in-vim-)**

## Disclaimer

En este articulo se da por hecho de que se tiene un conocimiento minimo de VIM,
entendimiento del modo `NORMAL`, `INSERT`, `VISUAL`, `V-LINE` e `(INSERT)`, la
key leader, y una nocion de los 'objetos' en VIM, en caso contrario se recomienda
ver la documentacion y tours de VIM con `:h`.

La distribucion de mi teclado es 'English (US)', es desde donde mas facilidad he
sentido para acceder a los simbolos que utilizo con frecuencia intentando seguir
las reglas de la mecanografia.

No soy un experto en este editor, me considero ignorante de la gran mayoria de
cosas que lo componen, esta es solo una recopilacion de mi utilizacion actual
que me a servido para ser mas productivo al desarrollar software, tengo mucho
que aprender, recomiendaciones y consejos que adoptar.

## Key leader

He configurado la key leader de mi VIM para utilizarla con `<Space>` (tecla de
espacio), de esta forma tengo un acceso mas facil y rapido con ambas manos al
tener una posicion de mecanografia.

## Fast moving in the file

Al abrir un archivo en VIM, utilizo `<Ctrl>f` para bajar y `<Ctrl>b` para subir
en el archivo, avanzando y retrocediendo 'pagina entera' hasta encontrar lo que
me interesa... si requiero desplazarme por un bloque de codigo cuidadosamente
para visualizar a detalle lo que va sucediendo me muevo con `<Ctrl>d` para bajar
y `<Ctrl>u` para subir 'media pagina'... si se necesita un scroll mas preciso
utilizo `<Ctrl>e` para bajar suavemente el scroll y `<Ctrl>y` para subirlo,
`zz` para centrar la pantalla con el cursor.

Esta configuracion usa [leap.nvim](https://github.com/ggandor/leap.nvim) como
plugin tipo EasyMotion, presiono `s` para delante del cursor o `S` para detras
del cursor, y posteriormente 2 caracteres de donde quiero llegar, y luego la
tecla final indicada por [leap.nvim](https://github.com/ggandor/leap.nvim) para llegar.

### Before I used an EasyMotion

Si no usara un plugin del estilo de EasyMotion, la forma de movimiento era:

Una vez visualizo lo que necesito editar, paso a observar el numero de linea relativo,
pulso el numero y pulso `j` o `k` dependiendo en que direccion esta la linea. Ejemplo:

_La linea objetivo esta 17 lineas debajo de donde estoy_ : `17j`

_La linea objetivo esta 13 lineas debajo de donde estoy_ : `13k`

Al estar en la linea, uso `f` o `F` mas el caracter del lugar de la linea al que
quiero ir, seguido de los `;` necesarios para repetir la busqueda y avanzar, por
ejemplo: `fp;;` para ir a la tercera "p" de la linea.

_Nota: El punto es dejar de moverse por el archivo solo dejando presionado
`j` para bajar y `k` para subir, la idea es no hacer ningun movimiento innecesario,
no necesitas pasar linea por linea para subir o bajar en un archivo, no necesitas
esperar que el cursor llegue al borde de la pantalla para apenas entonces empezar
a hacer scroll._

## Actions in file

- Ir al inicio del archivo: `gg`

- Ir al final del archivo: `G`

- Ir a una linea: `{numeroLinea}G` | example: `32G`

- Ir a un porcentaje del archivo: `{porcentaje}%` | example: `50%`

- Buscar hacia delante: `/text`

- Buscar hacia atras: `?text`

- Ir al siguiente resultado buscado: `n`

- Ir al anterior resultado buscado: `N`

- Ir al ultimo lugar donde se uso el modo `INSERT`: `'[`

- Ir al ultimo lugar donde se uso el modo `VISUAL`: `'<`

- Marcar un lugar: `m{a-zA-Z}` (example: `mk`, crea la marca `k`)

- Ir a una marca: `'{a-zA-Z}` (example: `'k`, va a la marca `k`, las marcas en
  mayuscula pueden ir a otros archivos)

## Actions in line

- Inicio de la linea: `0`

- Fin de la linea: `$`

- Ir al primer caracter non-blank de la linea: `^`

- Columna del medio: `gm`

- Ir a caracter hacia la derecha: `f{caracter}` | example: `fg`, va a la 'g'
  hacia la derecha

- Ir a caracter hacia la izquierda: `F{caracter}` | example: `Fg`, va a la 'g'
  hacia la izquierda

- Ir a caracter antes a la derecha: `t{caracter}` | example: `ta`, va a un
  caracter antes de la 'a' a la derecha

- Ir a caracter antes a la izquierda: `T{caracter}` | example: `Ta`, va a un
  caracter antes de la 'a' a la izquierda

- Repetir busqueda de caracter: `;`

- Repetir busqueda de caracter en la direccion opuesta: `,`

- Ir al final de la siguiente palabra: `e`

- Ir al final de la palabra anterior: `ge`

## Other common actions

Utilizo `"+` para usar el "registro" del clipboard del sistema operativo, por
ejemplo, si quiero copiar una linea al portapapeles pulso `"+yy`, o solo `"+y`
para copiar lo seleccionado en modo `VISUAL`.

Utilizo `u` para retrocer un cambio, es el "`<Ctrl>z`" de VIM. Con `<Ctrl>r`
rehaces el ultimo cambio retrocedido.

Utilizo `%` para ir hacia el opuesto del caracter en el que este el cursor
(ejemplo: `()`, `{}`, `[]`), por ejemplo para ir al final de una funcion usaria
`f{` para ir al `{` de apertura y luego uso `%` para ir rapidamente al `}` de cierre.

Cuando utilizo alguna un "Ir a la definicion", o en general hago un "jump" que me
lleva hacia otro lado... Regreso a donde estaba anterior con `<Ctrl>o` y vuelvo a
donde estaba antes de regresar con `<Ctrl>i`

Utilizo `.` para repetir alguna edicion que hice en modo `INSERT`.

Utilizo `~` para alternar de Uppercase a Lowercase un caracter, puede usarse en
modo `VISUAL` para alternar mas de un caracter rapidamente.

Utilizo `ci"` o `ci(` para cambiar todo el contenido interno de las comillas
(`""`) o parentesis (`()`) en las que me encuentro y editar, esto aplica para
tambien para cualquier caracter con caracter de cierre, la `c` puede ser sustituida
con un `d` para borrar o `y` para copiar... Por ejemplo `yi"`.

Utilizo `*` para buscar la palabra en donde esta el cursor hacia delante, y `#` para
buscar hacia atras, sin tener que escribirla en un busqueda con `/` o `?`.

Utilizo `:r <file>` para insertar el contenido de un archivo en la posicion del cursor.

Utilizo `<Ctrl>g` para visualizar el nombre completo del archivo actual.

Utilizo `:cc` para ir al primer error de la "quickfix list"

Utilizo `g]` para ir al siguiente error de la "quickfix list"

Utilizo `g[` para ir al error anterior de la "quickfix list"

Utilizo `o` para alternar la posicion del cursos en lo resaltado en modo `VISUAL`

Utilizo `:Bo` para ejecutar un script que cierra todos los archivos del buffer y
tabs, menos el archivo actual. Esto optimiza mucho el rendimiento si he abierto
varios archivos.

## File navigation

Para trabajar con varios archivos uso principalmente buffers y splits, para crear
un split vertical pulso `:vsplit` y `:split` para un split horizontal, entre los
splits me muevo con `<Ctrl>w{direction}`, la "`direction`" es la tecla de movimiendo
hacia donde esta el split... `h`, `j`, `k`, `l`.

Como explorador de archivos uso [`nvim-tree.lua`](https://github.com/kyazdani42/nvim-tree.lua):

- Abrir y cerrar `nvim-tree`: `<Space>n`

- Abrir `nvim-tree` posicionado en el archivo actual: `<Space>e`

- Abrir y cerrar una carpeta en `nvim-tree` estando sobre ella: `<Enter>`

- Cerrar una carpeta en `nvim-tree`: `x`

- Actualizar arbol: `R`

- Abrir un archivo en split vertical: `v`

- Abrir un archivo en split horizontal: `s`

- Ver previsualizacion del archivo: `<Tab>`

- Copiar un archivo: `c`

- Pegar un archivo: `p`

- Copiar nombre del archivo: `y`

- Copiar path relativo del archivo: `Y`

- Copiar path absoluto del archivo: `gy`

- Crear un archivo/carpeta en `nvim-tree`: `a`

- Borrar un archivo/carpeta en `nvim-tree`: `d`

- Renombrar/mover un archivo/carpeta en `nvim-tree`: `r`

- Navegar a la carpeta padre `nvim-tree`: `-`

- Abrir un archivo en una tab: `t`

---

Para hacer una busqueda de archivo utilizo el paquete [`telescope.nvim`](https://github.com/nvim-telescope/telescope.nvim):

- Abrir `telescope.nvim`: `<Space>f`

- Ver el buffer con `telescope.nvim`: `<Space>b`

- Buscar texto en archivos con `telescope.nvim`: `<Space>j`

- Salir de `telescope.nvim` (en modo `INSERT`): `<Ctrl>c`

- Salir de `telescope.nvim` (en modo `NORMAL`): `<Esc>` | `q`

- Moverse por los archivos de la lista (en modo `NORMAL`): `j` | `k`

- Moverse por los archivos de la lista (en modo `INSERT`): `<Ctrl>p` | `<Ctrl>n`

- Abrir archivo en split (en modo `INSERT`): `<Ctrl>s`

- Abrir archivo en split (en modo `NORMAL`): `s`

- Abrir archivo en vsplit (en modo `INSERT`): `<Ctrl>v`

- Abrir archivo en vsplit (en modo `NORMAL`): `v`

- Borrar del buffer: `<Ctrl>d`

- Seleccionar archivos: `<Tab>`

- Volver a modo `INSERT` para cambiar busqueda: `i`

- Abrir archivo en nueva tab (en modo `INSERT`): `<Ctrl>t`

- Abrir archivo en nueva tab (en modo `NORMAL`): `t`

## Tricks with Tabs and Splits

Con `<Ctrl-w>r` puedes rotar splits.

Con `<Ctrl-w>{H,J,K,L}` puedes mover un split.

Con `<Ctrl-w>=` haces que todos los splits tengan el mismo tamaño.

Con `<Ctrl-w>-` puedes decrementar el alto del split, con `<Ctrl-w>+` puedes
incrementar el alto del split.

Con `<Ctrl-w><` puedes decrementar ancho del split, con `<Ctrl-w>>` puedes
incrementar el ancho del split.

Con `<Ctrl-w>_` puedes darle todo el alto al split, con `<Ctrl-w>|` puedes
darle todo el ancho al split.

Con `20<Ctrl-w>_` puedes darle 20 de alto al split.
Con `20<Ctrl-w>|` puedes darle 20 de ancho al split.

## Replace text

En VIM utilizamos `:s` para reemplazar texto, este comando acepta expresiones regulares.

La sintaxis basica seria : `:{range}s/text/other/{options}`

Por ejemplo... con `:s/hello/world` reemplazaria la primera coincidencia de
"`hello`" con "`world`" en la linea.

Si quisiera cambiar todas la coincidencia de la linea utilizaria a `option`
"`g`", por ejemplo... `:s/hello/world`, asi reemplazaria todas las coincidencias
de "`hello`" con "`world`" en la linea.

Para ejecutar estos reemplazos en todo el archivo se utilizaria el `range` "`%`"
y la `option` "`g`", por ejemplo... `:%s/hello/world/g`.

Para repetir un reemplazo pero con un distinto rango y opciones... basta con
ponerlo sin la sentencia de busqueda ni la sentencia de reemplazo,
por ejemplo: `:%sgc`, eso repetiria el ultimo reemplazo que hice pero en todo
el archivo y confirmando cada cambio.

Hay mas `options` y `ranges` con los que se puede jugar de manera muy
interesante... dejare algunas combinaciones frecuentes que uso.

## Common replacement combinations

- Reemplazar todas las coincidencias pero confirmando cada cambio: `option: gc` |
  example: `:%s/text/other/gc`

- Reemplazar usando un rango de lineas absolutas: `:{line},{line}s/text/other/g`
  | example: `:15,20s/text/other/g`

- Reemplazar usando un rango de lineas relativas a la actual:
  `:.,.{+/-}{lines}s/text/other/g` | example: `:.,.+3s/text/other/g`

- Repetir el ultimo reemplazo pero sin las opciones: `&` o `:s`

- Repetir el ultimo reemplazo pero con opciones: `:s{options}` | example: `:sgc`

- Repetir el ultimo reemplazo en todo el archivo pero con opciones:
  `:%s{options}` | example: `:%sgc`

## Macros

Las macros en VIM son un superpoder fantanstico que no se aprovechar del todo
pero en algun momento le he sacado algo de provecho.

En VIM la cosa va de repeticiones, y las macro son un metodo de hacerlo a gran
escala, con el `.` podemos repetir el ultimo realizado en modo `INSERT`... pero
aveces necesitamos mas que eso. Primero recordemos como grabajar y ejecutar macros:

Empezamos a grabar una macro en algun registro con `q{a-z}`... por ejemplo: `qq`,
una vez realizamos las acciones que queramos repetir detenemos la "grabacion"
pulsando `q` de nuevo.

Podemos ejecutar nuestra macro con `@{a-z}`... por ejemplo: `@q`.

Y con `@@` podemos ejecutar rapidamente la ultima macro ejecutada!

## Small example of use

Pasando de esto:

    Lorem Ipsum is simply dummy text of the printing
    Lorem Ipsum has been the industry's
    It was popularised in the 1960s
    Contrary to popular belief, Lorem Ipsum is not simply random text. It has roots in a piece of classical
    majority have suffered alteration in some form, by
    like readable English. Many desktop
    This book is a treatise on the theory of ethics, very popular during the Renaissance
    All the Lorem Ipsum generators on the Internet tend
    The standard chunk of Lorem Ipsum used since the 1500s is reproduced below for those interested
    like readable English. Many desktop
    This book is a treatise on the theory of ethics, very popular during the Renaissance
    All the Lorem Ipsum generators on the Internet tend
    Contrary to popular belief, Lorem Ipsum is not simply random text. It has roots in a piece of classical

A esto, usando macros:

    "Lorem" : "Ipsum is simply dummy text of the printing",
    "Lorem" : "Ipsum has been the industry's",
    "It" : "was popularised in the 1960s",
    "Contrary" : "to popular belief, Lorem Ipsum is not simply random text. It has roots in a piece of classical",
    "majority" : "have suffered alteration in some form, by",
    "like" : "readable English. Many desktop",
    "This" : "book is a treatise on the theory of ethics, very popular during the Renaissance",
    "All" : "the Lorem Ipsum generators on the Internet tend",
    "The" : "standard chunk of Lorem Ipsum used since the 1500s is reproduced below for those interested",
    "like" : "readable English. Many desktop",
    "This" : "book is a treatise on the theory of ethics, very popular during the Renaissance",
    "All" : "the Lorem Ipsum generators on the Internet tend",
    "Contrary" : "to popular belief, Lorem Ipsum is not simply random text. It has roots in a piece of classical",

Macro usada: `qqI"<Esc>f<Space>i"<Space>:<Esc>la"<Esc>A"<Esc>jq` (recuerde que
en mi configuracion el `<Esc>` puede realizarse presionando `jk` o `kj`)

Basicamente fue... empezar a grabar la macro, hacer la modificacion deseada lo
mas repetible posible en donde sea que se ejecute, y finalizar moviendose hacia
algun lugar para solo ejecutar la macro una vez y luego seguir ejecutando con
`@@` (o el numero de veces que quieres repetir la ultima macro, por ejemplo... `12@@`)

Las macros son increiblemente utilices pero complicadas de "pensar" al inicio,
pueden usarse para cualquier accion repetitiva, este ejemplo que utilice fue un
caso util al momento de usar JSON (por ejemplo), imagina que en vez de ser solo
12 lineas que debes modificar de esa manera, son 30 o mas, aun siendo 'veloz' en
VIM o 'pecar' usando el mouse tomaria mucho mas tiempo que hacer una macro.

## Records

Puedes usar registros en VIM usando `"` y el nombre del registro, por ejemplo
`"j` accede al registro `j`, de esta manera puedes guardar texto como si tuvieras
muchos portapapeles, por ejemplo... `"jyy` copia una linea al registro `j` y con
`"jp` pegarias el contenido del registro `j`, en modo `INSERT` puedes pulgar
`<Ctrl-r>{record}` para insertar el contenido de un registro directamente, por
ejemplo... en modo `INSERT` : `<Ctrl-r>j` insertaria el registro `j` en donde
estemos. (El registro del portapapeles del sistema operativo es `+`)

Podemos asignar un registro con cualquier letra desde a-zA-Z, en VIM hay registros
especiales como el `=` el cual nos permite hacer operaciones matematicas, podemos
usarlo con `<Ctrl-r>` mientras estamos en modo `INSERT`.

Por ejemplo, podriamos estar escribiendo y pulgar `<Ctrl-r>=10+25<Enter>` y se
insertaria el resultado, podemos tambien jugar con las variables y otros registros,
por ejemplo...

En modo `NORMAL`: `:let i=10<Enter>`, con eso creamos una variable `i` con valor
`10`, y en modo `INSERT` podriamos pulgar `<Ctrl-r>=i+15<Enter>`.

Se pueden hacer muchas cosas productivas con esto y Macros.

## Indent lines

Programando suelo necesitar indentar lineas, en VIM puedo hacerlo con `<<` y `>>`
para cada direccion, tambien puede usarse en modo `VISUAL` para aplicar a varias
lineas.

Se puede recurrir al indentado automatico por defecto que VIM considere correcto
con `=`, puede usarse en un `range` y en modo `VISUAL`, por ejemplo... `gg=G`
indentaria todo el archivo.

## Suspend Vim

Con `<Ctrl-z>` puedes suspender el estado actual de VIM y volver a la terminal,
esto lo dejara como proceso de background... al igual que con otros procesos en
background puedes volver a el ejecutando `%` en la terminal

---

## 🔥 Programming in Vim 🔥

Este entorno usa [LunarVim](https://www.lunarvim.org/), por lo cual el comando
editor es `lvim`.

### Common actions

- Ir a la derecha en el bufferline: `<Shift>l`
- Ir a la izquierda en el bufferline: `<Shift>h`
- Pinear en el bufferline: `<Ctrl>p`
- Cerrar archivo del bufferline: `<Ctrl>c`
- Mover archivo a la derecha en el bufferline: `<Ctrl>l`
- Mover archivo a la izquierda en el bufferline: `<Ctrl>h`
- Ir a un archivo del bufferline con su posicion: `<Alt>{number}` | example: `<Alt>2`

- Comentar lineas toggle en modo `VISUAL`: `gc`
- Comentar bloque toggle en modo `VISUAL`: `gb`
- Comentar linea toggle en modo `NORMAL`: `gcc`
- Comentar bloque toggle en modo `NORMAL`: `gbc`

- Abrir link/path: `gx`
- Ir a definicion: `gd`
- Ir a la definicion del tipo: `gt`
- Ir a las implementaciones de una interface o metodo: `gi`
- Ir a las referencias de la funcion: `gr`

- Ver seccion de declaracion en una ventana flotante: `gp`
- Ver documentacion/hover de un simbolo/codigo: `K`
- Saltar a la ventana flotante de documentacion/hover: `KK`
- Scroll en el autocompletado: `<Ctrl>f` | `<Ctrl>d`

- Refactorizar nombre: `<leader>rn`
- Herramientas de refactorizacion: `<leader>rr`
- Mostrar code actions: `<leader>a`
- Mostrar code lens: `<leader>s`

- Abrir quickfix list: `<leader>l`
- Siguiente error de la quickfix list: `g]` | `:cn`
- Anterior error de la quickfix list: `g[` | `:cp`
- Abrir resultado de la quickfix list en un split: `s`
- Abrir resultado de la quickfix list en un vsplit: `v`
- Abrir resultado de la quickfix list en una tab: `t`

- Regresar un `jump`: `<Ctrl>o`
- Avanzar al `jump`: `<Ctrl>o`

- Agregar `tags` a una estructura: `<leader>gat`
- Eliminar `tags` de una estructura: `<leader>grt`

- Ejecutar prueba de covertura: `:GoCoverage`
- Coverage toggle del codigo: `:GoCoverage -t`
- Ejecutar tests: `:GoTest`

- Llamar a `:GoBuild`: `<leader>gj`
- Llamar a `golangci-lint`: `<leader>gl`

- Buscar archivo: `<leader>f`
- Buscar archivo en buffer: `<leader>b`
- Buscar texto en archivos: `<leader>j`

- Ir al extremo opuesto de comillas, parentesis, llaves, corchetes, etc...: `%`

- Cerrar todos los archivos en buffer menos el actual: `:Bo`

- Copiar path actual: `:Cpath`

- Actualizar contenido del archivo actual: `:e`

- Alternar numeros de linea relativas: `:set rnu!`

- Guardar sin formatear: `:noa w`

- Cambiar tamaño del `nvim-tree`: `:NvimTreeResize 50`

- Previsualizar Markdown: `:MarkdownPreview`

- Buscar espacios en blanco al final de lineas: `:SearchWhitespace`
- Eliminar espacios en blanco al final de lineas: `:StripWhitespace`

- Recargar linters/null-ls: `:LinterRestart`
- Desactivar linters/null-ls: `:LinterDisable`
- Activar linters/null-ls: `:LinterEnable`
- Detener linters/null-ls: `:LinterStop`

- Desactivar el resaltado de busqueda: `:nohl`

- Cambiar a modo LightTheme: `:LightTheme`
- Cambiar a modo DarkTheme: `:DarkTheme`

- Cerrar fold actual: `zc`
- Abrir fold actual: `zo`
- Toggle fold actual: `za`
- Cerrar todos los folds actuales (recursivamente): `zC`
- Abrir todos los folds actuales (recursivamente): `zO`
- Ir al siguiente fold: `zj`
- Ir al anterior fold: `zk`
- Abrir todos los folds del nivel de indentacion actual: `zr`
- Abrir todos los folds: `zR`
- Cerrar todos los folds: `zM`

## vim-surround

Esta configuracion incluye [vim-surround](https://github.com/tpope/vim-surround),
para la utilizacion del mismo recomiendo leer su [README.markdown](https://github.com/tpope/vim-surround/blob/master/README.markdown)
ya que explica todo lo fundamental
