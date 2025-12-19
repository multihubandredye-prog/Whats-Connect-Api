export default {
    name: 'AppLoginWithCode',
    props: {
        connected: {
            type: Array,
            default: [],
        }
    },
    watch: {
        connected: function(val) {
            if (val) {
                // reset form
                this.phone = '';
                this.pair_code = null;

                $('#modalLoginWithCode').modal('hide');
            }
        },
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
                if (this.connected) throw Error('Você já está conectado');

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
            <div class="header">Login Com Código</div>
            <div class="description">
                Insira seu código de emparelhamento para fazer login e acessar seu dispositivo
            </div>
        </div>
    </div>
    
    <!--  Modal Login  -->
    <div class="ui small modal" id="modalLoginWithCode">
        <i class="close icon"></i>
        <div class="header">
            Obtendo código de pareamento
        </div>
        <div class="content">
            <div class="ui message info">
                <div class="header">Como emparelhar?</div>
                <ol>
                    <li>Abra seu Whatsapp</li>
                    <li>Vincular um dispositivo</li>
                    <li>Link com código de pareamento</li>
                </ol>
            </div>
            
            <div class="ui form">
                <div class="field">
                    <label>Phone</label>
                    <input type="text" v-model="phone" placeholder="Type your phone number"
                        @keyup.enter="handleSubmit" :disabled="submitting">
                    <small>Enter to submit</small>
                </div>
            </div>
            
            <div class="ui grid" v-if="pair_code">
                <div class="ui two column centered grid">
                    <div class="column center aligned">
                        <div class="header">Código de pareamento</div>
                        <p style="font-size: 32px">{{ pair_code }}</p>
                        
                    </div>
                </div>
            </div>
        </div>
    </div>
    `,
};