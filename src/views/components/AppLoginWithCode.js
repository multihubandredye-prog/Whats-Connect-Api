export default {
    name: 'AppLoginWithCode',
    props: {
        loggedIn: {
            type: Boolean,
            default: false,
        }
    },
    data: () => {
        return {
            phone: '',
            submitting: false,
            pair_code: null,
        };
    },
    methods: {
        async openModal() {
            try {
                if (this.loggedIn) throw Error('Você já está logado.');

                $('#modalLoginWithCode').modal({
                    onApprove: function() {
                        return false;
                    },
                }).modal('show');
            } catch (err) {
                showErrorInfo(err);
            }
        },
        async handleSubmit() {
            if (this.submitting) return;
            try {
                this.submitting = true;
                const { data } = await http.get(`/app/login-with-code`, {
                    params: {
                        phone: this.phone,
                    },
                });
                this.pair_code = data.results.pair_code;
            } catch (err) {
                if (err.response) {
                    showErrorInfo(err.response.data.message);
                }else{
                    showErrorInfo(err.message);
                }
            } finally {
                this.submitting = false;
            }
        },
    },
    template: `
    <div class="green card" @click="openModal" style="cursor: pointer">
        <div class="content">
            <a class="ui teal right ribbon label">App</a>
            <div class="header">Entrar com Código</div>
            <div class="description">
                Insira seu código de pareamento para entrar e acessar seus dispositivos.
            </div>
        </div>
    </div>
    
    <!-- Modal de Login  -->
    <div class="ui small modal" id="modalLoginWithCode">
        <i class="close icon"></i>
        <div class="header">
            Obtendo Código de Pareamento
        </div>
        <div class="content">
            <div class="ui message info">
                <div class="header">Como parear?</div>
                <ol>
                    <li>Abra seu Whatsapp</li>
                    <li>Conectar um aparelho</li>
                    <li>Conectar com código de pareamento</li>
                </ol>
            </div>
            
            <div class="ui form">
                <div class="field">
                    <label>Telefone</label>
                    <input type="text" v-model="phone" placeholder="Digite seu número de telefone"
                        @keyup.enter="handleSubmit" :disabled="submitting">
                    <small>Pressione Enter para enviar</small>
                </div>
            </div>
            
            <div class="ui grid" v-if="pair_code">
                <div class="ui two column centered grid">
                    <div class="column center aligned">
                        <div class="header">Código de Pareamento</div>
                        <p style="font-size: 32px">{{ pair_code }}</p>
                        
                    </div>
                </div>
            </div>
        </div>
    </div>
    `,
};