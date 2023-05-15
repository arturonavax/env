# How do I use Terminal?

Desarrollo y trabajo utilizando 100% la terminal de Linux (menos el Navegador
Web), editor de codigo, bases de datos, monitorizacion de sistema.

La GUI de terminal que utilizo es indiferente a mi flujo de trabajo, suelo usar
la [Alacritty](https://github.com/alacritty/alacritty) porque tiene aceleracion
por GPU y se me hizo mas facil de transportar su configuracion, mi flujo de
trabajo se divide en :

- Terminal GUI: [`alacritty`](https://github.com/alacritty/alacritty)

  - Nerd Font: [Caskaydia Cove](https://github.com/ryanoasis/nerd-fonts/tree/master/patched-fonts/CascadiaCode)

- Shell: `zsh`
- Terminal Multiplexor: `tmux`
- Code editor: `lvim` ([LunarVim](https://www.lunarvim.org/))

Para aumentar la velocidad del teclado seteo el delay y repeat rate para el
servidor grafico con:

```sh
xset r rate 200 30
```

Al encender mi maquina, abro la `alacritty`, voy a la carpeta de mi proyecto,
y presiono `<F11>` para trabajar con la terminal en pantalla completa.

## Alacritty

Utilizo `alacritty` como GUI de terminal, por la aceleracion por GPU que ofrece
y la facil configuracion por un archivo .yml, esto ultimo me permite transportar
facilmente la configuracion.

Esta terminal la tengo configurada sin bordes de ventana por decoracion, muevo y
manejo la ventana por medio de los atajos de `<Super>` (Window Key)

- Mover la ventana con el mouse: `<Super>`

- Ventana a pantalla completa: `<F11>`

- Minimizar ventana: `<Super>h`

- Maximizar ventana: `<Super><UpArrow>`

- Maximizar ventana a la derecha: `<Super><RightArrow>`

- Maximizar ventana a la izquierda: `<Super><LeftArrow>`

- Tamaño original de ventana: `<Super><DownArrow>`

## Zsh

Utilizo `zsh` como Shell en reemplazo a `bash`, para el autocompletado y plugins
visuales con `Oh My Zsh!`.

## Tmux

Utilizo `tmux` para multiplexar mi terminal con varios splits y paneles, he
cambiado la `"Key leader"` a `<Ctrl>a` en reemplazo a `<Ctrl>b`.

- Crear split horizontal: `<Ctrl>a%`

- Crear split vertical: `<Ctrl>a"`

- Hacer toggle zoom de un split: `<Ctrl>az`

- Crear nuevo panel: `<Ctrl>ac`

- Ir al siguiente panel: `<Ctrl>an`

- Ir al panel anterior: `<Ctrl>ap`

- Moverte entre splits (con movimientos de Vim): `<Ctrl>a{h, j, k, l}` |
  example: `<Ctrl>al`

- Redimensionar split (con movimientos de Vim): `<Ctrl>a{<, +, -, >}` |
  example: `<Ctrl>a>`

- Modo de movimiento Vi: `<Ctrl>a[`

- Renombrar panel: `<Ctrl>a,`

- Rotar paneles: `<Ctrl>a<Space>`

- Abrir link seleccionado/subrayado: `x`

- Abrir link solo con click: `<Shift><Click>`

Se puede redimensionar splits con el mouse, arrastrando las lineas separadoras.

Al seleccionar texto con el mouse y dejarlo presionado, se puede presionar `y`
para copiar lo seleccionado o `x` para abrir el link seleccionado.

## Common commands/actions

- Copiar path actual al portapapeles: `pwd | xclip -selection clipboard`

- Abrir archivo o carpeta con su programa por defecto: `xdg-open <PATH/NAME>`

- Buscador de comandos: `<Ctrl>r`

- "Flecha arriba": `<Ctrl>p`

- "Flecha abajo": `<Ctrl>n`

- Ir al inicio de la linea: `<Ctrl>a <Ctrl>a` (Doble, por usar Tmux)

- Ir al final de la linea: `<Ctrl>e`

- Borrar desde el cursor hasta el inicio de la linea: `<Ctrl>u`

- Borrar desde el cursor hasta el final de la linea: `<Ctrl>k`

- Borrar palabra antes del cursor: `<Ctrl>w`

- Borrar palabra siguiente del cursor: `<Alt>d`

- Ir a siguiente palabra: `<Alt>f`

- Ir a anterior palabra: `<Alt>b`

- Ir a siguiente letra: `<Ctrl>f`

- Ir a anterior letra: `<Ctrl>b`

- Suspender proceso, ponerlo en background: `<Ctrl>z`

- Recuperar proceso en background: `fg` | `%`

- Ver procesos en background: `jobs`
