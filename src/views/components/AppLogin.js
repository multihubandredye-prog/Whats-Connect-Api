export default {
    name: 'AppLogin',
    props: {
        loggedIn: {
            type: Boolean,
            default: false,
        },
    },
    data() {
        return {
            login_link: '',
            login_duration_sec: 0,
            countdown_timer: null,
        }
    },
    methods: {
        async openModal() {
            try {
                if (this.loggedIn) throw Error('Você já está logado.');

                await this.submitApi();
                $('#modalLogin').modal({
                    onApprove: function () {
                        return false;
                    },
                    onHidden: () => {
                        this.stopCountdown();
                    }
                }).modal('show');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            try {
                // Parar a contagem regressiva existente antes de fazer uma nova solicitação
                this.stopCountdown();
                
                let response = await window.http.get(`app/login`)
                let results = response.data.results;
                this.login_link = results.qr_link;
                this.login_duration_sec = results.qr_duration;
                
                // Iniciar contagem regressiva após chamada de API bem-sucedida
                this.startCountdown();
            } catch (error) {
                if (error.response) {
                    throw Error(error.response.data.message)
                }
                throw Error(error.message)
            }
        },
        startCountdown() {
            // Limpar qualquer timer existente
            this.stopCountdown();
            
            this.countdown_timer = setInterval(() => {
                if (this.login_duration_sec > 0) {
                    this.login_duration_sec--;
                } else {
                    // Atualização automática quando a contagem regressiva chega a 0
                    this.autoRefresh();
                }
            }, 1000);
        },
        stopCountdown() {
            if (this.countdown_timer) {
                clearInterval(this.countdown_timer);
                this.countdown_timer = null;
            }
        },
        async autoRefresh() {
            try {
                console.log('Código QR expirado, atualizando automaticamente...');
                await this.submitApi();
            } catch (error) {
                console.error('Falha na atualização automática:', error);
                this.stopCountdown();
                showErrorInfo(error);
            }
        }
    },
    beforeUnmount() {
        // Limpar o timer quando o componente é destruído
        this.stopCountdown();
    },
    template: `
    <div class="green card" @click="openModal" style="cursor: pointer">
        <div class="content">
            <a class="ui teal right ribbon label">App</a>
            <div class="header">Entrar</div>
            <div class="description">
                Escaneie seu código QR para acessar todos os recursos da API.
            </div>
        </div>
    </div>
    
    <!--  Modal Login  -->
    <div class="ui small modal" id="modalLogin">
        <i class="close icon"></i>
        <div class="header">
            Entrar no Whatsapp
        </div>
        <div class="image content">
            <div class="ui medium image">
                <img :src="login_link" alt="Código QR para Login">
            </div>
            <div class="description">
                <div class="ui header">Por favor, escaneie para conectar</div>
                <p>Abrir Configurações > Aparelhos Conectados > Conectar um Aparelho</p>
                <div style="padding-top: 50px;">
                    <i v-if="login_duration_sec > 0">O Código QR expira em {{ login_duration_sec }} segundos (atualização automática)</i>
                    <i v-else class="ui active inline">Atualizando Código QR...</i>
                </div>
            </div>
        </div>
        <div class="actions">
            <div class="ui approve positive right labeled icon button" @click="submitApi">
                Atualizar Código QR
                <i class="refresh icon"></i>
            </div>
        </div>
    </div>
    `
}