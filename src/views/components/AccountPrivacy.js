export default {
    name: 'AccountPrivacy',
    data() {
        return {
            data_privacy: null
        }
    },
    methods: {
        async openModal() {
            try {
                await this.submitApi();
                $('#modalUserPrivacy').modal('show');
                showSuccessInfo("Configurações de privacidade obtidas")
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            try {
                let response = await window.http.get(`/user/my/privacy`)
                this.data_privacy = response.data.results;
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            }
        },
    },
    template: `
    <div class="olive card" @click="openModal" style="cursor: pointer">
        <div class="content">
        <a class="ui olive right ribbon label">Conta</a>
            <div class="header">Minhas Configurações de Privacidade</div>
            <div class="description">
                Obtenha suas configurações de privacidade
            </div>
        </div>
    </div>
    
    <!--  Modal UserPrivacy  -->
    <div class="ui small modal" id="modalUserPrivacy">
        <i class="close icon"></i>
        <div class="header">
            Minha Privacidade
        </div>
        <div class="content">
            <ol v-if="data_privacy != null">
                <li>Quem pode adicionar a grupos: <b>{{ data_privacy.group_add }}</b></li>
                <li>Quem pode ver meu Visto por Último: <b>{{ data_privacy.last_seen }}</b></li>
                <li>Quem pode ver meu Status: <b>{{ data_privacy.status }}</b></li>
                <li>Quem pode ver meu Perfil: <b>{{ data_privacy.profile }}</b></li>
                <li>Confirmações de Leitura: <b>{{ data_privacy.read_receipts }}</b></li>
            </ol>
        </div>
    </div>
    `
}