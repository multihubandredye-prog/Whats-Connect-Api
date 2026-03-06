export default {
    name: 'GroupSetName',
    data() {
        return {
            loading: false,
            groupId: '',
            name: '',
        }
    },
    methods: {
        openModal() {
            $('#modalGroupSetName').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isValidForm() {
            return this.groupId.trim() !== '' && this.name.trim() !== '' && this.name.length <= 25;
        },
        async handleSubmit() {
            if (!this.isValidForm() || this.loading) {
                return;
            }
            try {
                let response = await this.submitApi()
                showSuccessInfo(response)
                $('#modalGroupSetName').modal('hide');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let response = await window.http.post(`/group/name`, {
                    group_id: this.groupId,
                    name: this.name
                })
                this.handleReset();
                return response.data.message;
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            } finally {
                this.loading = false;
            }
        },
        handleReset() {
            this.groupId = '';
            this.name = '';
        },
    },
    template: `
    <div class="green card" @click="openModal" style="cursor: pointer">
        <div class="content">
            <a class="ui green right ribbon label">Grupo</a>
            <div class="header">Definir Nome do Grupo</div>
            <div class="description">
                Alterar o nome/título do grupo
            </div>
        </div>
    </div>
    
    <!--  Modal Group Set Name  -->
    <div class="ui small modal" id="modalGroupSetName">
        <i class="close icon"></i>
        <div class="header">
            Definir Nome do Grupo
        </div>
        <div class="content">
            <div class="ui info message">
                <i class="info circle icon"></i>
                Seu nome de exibição é o nome mostrado a outros no WhatsApp.
            </div>
            
            <form class="ui form">
                <div class="field">
                    <label>ID do Grupo</label>
                    <input v-model="groupId" type="text"
                           placeholder="120363024512399999@g.us"
                           aria-label="Group ID">
                </div>
                
                <div class="field">
                    <label>Nome do Grupo</label>
                    <input v-model="name" type="text"
                           placeholder="Insira o novo nome do grupo..."
                           maxlength="25"
                           aria-label="Group Name">
                    <small class="text">Máximo 25 caracteres. Comprimento atual: {{ name.length }}/25</small>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" 
                    :class="{'loading': this.loading, 'disabled': !this.isValidForm() || this.loading}"
                    @click.prevent="handleSubmit" type="button">
                Atualizar Nome
                <i class="edit icon"></i>
            </button>
        }
    </div>
    `
} 